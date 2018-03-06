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
// v2.0.0(180303) 画像urlの取得方法を変更
// v2.0.1(180303) increase download tool timeout
// v2.0.2(180306) uBlock Originを使うようにした

package main

import (
	"fmt"
	"time"
	"net/http"
	"net/http/cookiejar"
	"golang.org/x/net/publicsuffix"
	"errors"
	"os"
	"regexp"
	"math/rand"
	"sync"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"./tools"
	"./mmdl"
	"./img"
)


var VERSION = "v2.0.2(180306)"

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
	var err error

	err = tools.DownloadChromedriver()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	err = tools.DownloadUBlock()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	pageId, err := get_settings()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	dbroot := "./db/"
	if !mkdir(dbroot) {
		fmt.Printf("Can't create %s\n", dbroot)
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

	_, err = db.Exec(`
		create table if not exists pageinfo (
			title        text    not null
		)`)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}


	title, err := mmdl.Mmdl(pageId, db)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	err = tools.DownloadImageMagic()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	err = tools.DownloadPhantomJs()
	if err != nil {
		fmt.Printf("%v\n", err)
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

