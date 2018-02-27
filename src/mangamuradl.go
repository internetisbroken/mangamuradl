// mangamura downloader
//
// 上げたらvar VERSIONを更新すること
//
// 2018/02/17 最初の動くもの
// v1.0.1(180219) -cookie を追加(reCapture)
// v1.0.2(180220) getcookie に対応
// v1.0.3(180222) pdf作成機能追加,分割ページ対応
// v1.0.4(180226) base64画像に対応
// v1.1.0(180228) 安定化

package main

import (
	"fmt"
	"time"
	"net/http"
	"net/http/cookiejar"
	"golang.org/x/net/publicsuffix"
	"errors"
	"os"
	"os/exec"
	"regexp"
	"math/rand"
	"sync"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"./tools"
	"./conf"
	"./mm"
	"./img"
)


var VERSION = "v1.1.0(180228)"

func Setup() (err error) {
	rand.Seed(time.Now().UnixNano())
	http.DefaultClient.Timeout, _ = time.ParseDuration("40s")
	http.DefaultClient.Transport = http.DefaultTransport
	http.DefaultTransport.(*http.Transport).MaxIdleConns = 1
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 1
	http.DefaultTransport.(*http.Transport).ResponseHeaderTimeout, _ = time.ParseDuration("40s")

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return
	}
	http.DefaultClient.Jar = jar
	return
}


func get_settings() (pageId string, err error) {
	args := os.Args[1:]
	re_cookie := regexp.MustCompile(`(?i)^--?cookie=(\S+)`)
	re_0 := regexp.MustCompile(`p=(\w+)`)
	re_1 := regexp.MustCompile(`^(\d+)$`)
	for i := 0; i < len(args); i++ {
		m_cookie := re_cookie.FindStringSubmatch(args[i])
		if len(m_cookie) >= 2 {
			// deleted
		} else {
			m_0 := re_0.FindStringSubmatch(args[i])
			if len(m_0) >= 2 {
				pageId = m_0[1]
			} else {
				m_1 := re_1.FindStringSubmatch(args[i])
				if len(m_1) >= 2 {
					pageId = m_1[1]
				}
			}
		}
	}

	if pageId == "" {
		err = errors.New("page id required");
		return
	}

	return
}

func exec_getcookie() (err error) {
	err = tools.DownloadChromedriver()
	if err != nil {
		return
	}

	fmt.Printf("Chromeが立ち上がるので認証して下さい\n")
	out, err := exec.Command("getcookie").Output()
	fmt.Printf("\n%s\n", string(out))
	if err != nil {
		return
	} else {
		//fmt.Printf("認証できたら、コマンドを再実行して下さい\n")
	}
	return
}

func mkdir(dir string) bool {
	err := os.Mkdir(dir, 0777)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("%v\n", err)
			return false
		}
	}
	return true
}

func main() {

	fmt.Printf("version %s\n", VERSION)
	Setup()

	pageId, err := get_settings()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	dbroot := "./db/"
	if !mkdir(dbroot) {
		return
	}

	dbfile := fmt.Sprintf("%s/%s.sqlite3", dbroot, pageId)
	db, err := sql.Open("sqlite3", dbfile)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	defer db.Close()

	_, err = db.Exec(`
		create table if not exists page (
			id          integer  primary key autoincrement not null,
			path        text     not null,
			pagenum     integer  not null unique,
			req_status  integer  not null default 0,
			url         text     not null default "",
			is_frame    integer  not null default 0,
			is_blob     integer  not null default 0,
			blob_b64    text     not null default ""
		)`)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	title, err := mm.GetTitle(pageId)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	fmt.Printf("title[%v]\n", title)

	sioserver, err := mm.GetSocketioServer(pageId)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	//fmt.Printf("sioserver is %v\n", sioserver)

	err = mm.GetPageList(pageId, db)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	reconnect := true
	var auth bool
	var del_cookie bool
	var sid string
	req_max := 1
	ntries := 0
	var update_cookie bool
	var new_cookie string

	var recv_auth bool
	var recv_del bool
	var recv_update bool
	var recv_cookie string
	for {
		tx, err := db.Begin()
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}

		// index: req_status
		row := tx.QueryRow("select count(id) from page where req_status != 2")
		var requesting int
		err = row.Scan(&requesting)
		if err != nil {
			log.Fatal(err)
		}
		if requesting <= 0 {
			break
		}

		if reconnect {
			sid, err = mm.SioConnect(sioserver)
		}

		req_cnt, err := mm.SioRequest0(sioserver, sid, tx, req_max)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}
		if req_cnt == 0 {
			fmt.Printf("[FIXME] no data posted\n")
			break
		}

		var resp_cnt int
		resp_cnt, reconnect, auth, del_cookie, update_cookie, new_cookie, err = mm.SioRequest1(sioserver, sid, tx)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}

		if auth {
			recv_auth = true
		}
		if del_cookie {
			recv_del = true
		}
		if update_cookie {
			recv_update = true
			recv_cookie = new_cookie
		}

		if resp_cnt == 0 {
			time.Sleep(1 * time.Second)
			if false {
				fmt.Printf("req_cnt %d, resp_cnt %d, reconnect %v, auth %v, del_cookie %v, update_cookie %v, new_cookie %s\n",
					req_cnt, resp_cnt, reconnect, auth, del_cookie, update_cookie, new_cookie);
			}

			ntries++
			if ntries >= 6 {
				//conf.SetCookie("")
				break
			} else if ntries == 4 {
				if recv_update {
					recv_auth = false
					recv_del = false
					recv_update = false

					conf.SetCookie(recv_cookie)

				} else if recv_auth || recv_del {
					recv_auth = false
					recv_del = false
					recv_update = false

					fmt.Printf("Authentication required\n");
					err = exec_getcookie()
					if err != nil {
						fmt.Printf("Authentication error: %v\n", err)
						break
					}
				}
			}

		} else {
			ntries = 0
			recv_auth = false
			recv_del = false
		}


		if del_cookie {
			//UpdateIni("mangamuradl.ini", "acookie4", "")
		}

		if reconnect && (resp_cnt != req_cnt) {
			if req_max >= 50 {
				req_max = 24
			} else if req_max >= 24 {
				req_max = 7
			} else if req_max >= 7 {
				req_max = 1
			}
		} else if req_max < 10 {
			req_max = 24
		} else if req_max < 50 {
			req_max = 50
		} else if req_max <= 100 {
			req_max = 100
		}

		tx.Commit()
	} // loop

	err = tools.DownloadImageMagic()
	if err != nil {
		return
	}

	err = tools.DownloadPhantomJs()
	if err != nil {
		return
	}

	imgdir := "./img"
	if !mkdir(imgdir) {
		return
	}
	imgroot := fmt.Sprintf("%s/%s", imgdir, title)
	if !mkdir(imgroot) {
		return
	}

	rows, err := db.Query("select pagenum, url, is_frame, is_blob, blob_b64 from page where req_status == 2 order by pagenum")
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	defer rows.Close()

	wait := new(sync.WaitGroup)
	i := 0
	for rows.Next() {
		var pagenum int
		var url string
		var is_frame int
		var is_blob int
		var blob_b64 string
		err = rows.Scan(&pagenum, &url, &is_frame, &is_blob, &blob_b64)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}

		wait.Add(1)
		go func(root string, pagenum int, url string, is_frame, is_blob bool, base64str string) {
			err := img.DownloadImage(root, pagenum, url, is_frame, is_blob, base64str)
			if err != nil {
				fmt.Printf("%v\n", err)
			}
			wait.Done()
		}(imgroot, pagenum, url, is_frame != 0, is_blob != 0, blob_b64)

		i++
		if i % 8 == 7 {
			wait.Wait()
		}
	}
	wait.Wait()

	err = rows.Err()
	if err != nil {
		return
	}

	if true {
		pdfdir := "./pdf"
		if !mkdir(pdfdir) {
			fmt.Println("CreatePdf skipped")
		} else {
			err = img.CreatePdf(imgroot, fmt.Sprintf("%s/%s.pdf", pdfdir, title), db)
			if err != nil{
				fmt.Printf("CreatePdf: %v\n", err)
			}
		}
	}

	fmt.Printf("Done: %s\n", title);
	return
}

