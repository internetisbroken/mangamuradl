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
// v2.0.3(180310) zip作成機能追加
// v2.0.4(180312) fix: base64 error case
// v2.0.5(180315) fix create zip/pdf
// v2.0.6(180316) support webp
//
// 上げたらvar VERSIONを更新すること

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
	"github.com/sclevine/agouti"
	"./tools"
	"./mmdl"
	"./img"
	"./conf"
)

var VERSION = "v2.0.6(180316)"

func help() {
	fmt.Printf(`
mangamuradl [Options] PageId...

PageId:
  http://mangamura.org/?p=123456789
  123456789

Options:
  -h --help    print help
  -p --pdf     create pdf ON
  -P --no-pdf  create pdf OFF
  -z --zip     create zip ON
  -Z --no-zip  create zip OFF
`)
	os.Exit(0)
}


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

func argparse() (pageId string, err error) {
	args := os.Args[1:]

	type ArgCheck struct {
		re *regexp.Regexp
		cb func([]string)
	}

	chkList := []ArgCheck{
		ArgCheck{regexp.MustCompile(`(?i)^(?:--?|/)(?:h|help|\?)$`), func(ma []string) {
			help()
		}},
		ArgCheck{regexp.MustCompile(`^--?(?:P|(?i:no-pdf))$`), func(ma []string) {
			conf.SetPdf(false)
		}},
		ArgCheck{regexp.MustCompile(`^--?(?:p|(?i:pdf))$`), func(ma []string) {
			conf.SetPdf(true)
		}},
		ArgCheck{regexp.MustCompile(`^--?(?:Z|(?i:no-zip))$`), func(ma []string) {
			conf.SetZip(false)
		}},
		ArgCheck{regexp.MustCompile(`^--?(?:z|(?i:zip))$`), func(ma []string) {
			conf.SetZip(true)
		}},
		ArgCheck{regexp.MustCompile(`p=(\w+)|^(\d+)$`), func(ma []string) {
			if len(ma) >=2 && ma[1] != "" {
				pageId = ma[1]
			} else if len(ma) >=3 && ma[2] != "" {
				pageId = ma[2]
			}
		}},
	}

	NEXT_ARG: for _, v := range args {
		for _, chk := range chkList {
			m := chk.re.FindStringSubmatch(v)
			if len(m) >= 1 {
				chk.cb(m)
				continue NEXT_ARG
			}
		}
		fmt.Printf("Unknown option: %v\n", v)
		help()
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

var dbVersion = "180316"
func checkTable(db *sql.DB) (ok bool) {

	stmt, err := db.Prepare("select v from kv where k = ?")
	if err != nil {
		return
	}
	defer stmt.Close()
	var version string
	err = stmt.QueryRow("DB_VERSION").Scan(&version)
	if err != nil {
		return
	}

	if version != "" && version == dbVersion {
		ok = true
	}

	return
}

func createTable(db *sql.DB) (err error) {

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
		return
	}

	_, err = db.Exec(`
		create table if not exists kv (
			k  text  not null unique,
			v  text  not null
		)`)
	if err != nil {
		return
	}

	stmt, err := db.Prepare("insert or ignore into kv(k, v) values(?, ?)")
	if err != nil {
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec("DB_VERSION", dbVersion)
	if err != nil {
		return
	}

	return
}

func main() {
	var err error
	fmt.Printf("version %s\n", VERSION)

	Setup()

	pageId, err := argparse()
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
	var db *sql.DB
	db, err = sql.Open("sqlite3", dbfile)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	if (! checkTable(db)) {
		db.Close()
		fmt.Printf("Deleting old database: %s\n", dbfile)
		err = os.Remove(dbfile)
		if err != nil {
			fmt.Printf("%v\nPlease delete %s manually\n", err, dbfile)
			return
		}
		db, err = sql.Open("sqlite3", dbfile)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}
	}
	defer db.Close()

	err = createTable(db)
	if err != nil {
		return
	}

	// Get Image url list
	title, err := mmdl.Mmdl(pageId, db)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	_, err = tools.GetConvert()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	_, err = tools.GetPhantomjs()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	imgdir := "./img"
	if !mkdir(imgdir) {
		fmt.Printf("mkdir: Can't create %s\n", imgdir)
		return
	}
	imgroot := fmt.Sprintf("%s/%s", imgdir, title)
	if !mkdir(imgroot) {
		fmt.Printf("mkdir: Can't create %s\n", imgroot)
		return
	}

	// Non-frame
	err = func() (err error) {
		rows, err := db.Query(
			"select pagenum, url, is_blob, blob_b64 from page where req_status == 2 and is_frame <> 1 order by pagenum")
		if err != nil {
			return
		}
		defer rows.Close()

		wait := new(sync.WaitGroup)
		i := 0
		for rows.Next() {
			var pagenum int
			var url string
			var is_blob int
			var blob_b64 string
			err = rows.Scan(&pagenum, &url, &is_blob, &blob_b64)
			if err != nil {
				return
			}

			wait.Add(1)
			go func(root string, pagenum int, url string, is_blob bool, base64str string) {
				err := img.DownloadImage(root, pagenum, url, false, is_blob, base64str)
				if err != nil {
					fmt.Printf("%v\n", err)
				}
				wait.Done()
			}(imgroot, pagenum, url, is_blob != 0, blob_b64)

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
		return
	}()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	// framed page
	err = func() (err error) {
		rows, err := db.Query(
			"select pagenum, url, is_frame from page where req_status == 2 and is_frame == 1 order by pagenum")
		if err != nil {
			return
		}
		defer rows.Close()

		var opened bool
		var driver *agouti.WebDriver
		var page *agouti.Page
		for rows.Next() {
			var pagenum int
			var url string
			var is_frame int
			err = rows.Scan(&pagenum, &url, &is_frame)
			if err != nil {
				return
			}

			if ex, _ := img.FindImageByNumber(imgroot, pagenum); ex == false {
				// download if not saved
				if (! opened) {
					opened = true
					// start browser
					//driver, page, err = tools.StartChrome()
					driver, page, err = tools.StartPhantomjs()
					if err != nil {
						return
					}
					defer driver.Stop()
				}

				_, e := img.DownloadFrameImage(imgroot, pagenum, url, page)
				if e != nil {
					err = e
					return
				}
			}
		}

		err = rows.Err()
		if err != nil {
			return
		}
		return
	}()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	if conf.IsPdf() {
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
	if conf.IsZip() {
		zipdir := "./zip"
		if !mkdir(zipdir) {
			fmt.Println("CreateZip skipped")
		} else {
			err = img.CreateZip(imgroot, fmt.Sprintf("%s/%s.zip", zipdir, title), db)
			if err != nil{
				fmt.Printf("CreateZip: %v\n", err)
			}
		}
	}

	fmt.Printf("Done: %s\n", title);
	return
}

