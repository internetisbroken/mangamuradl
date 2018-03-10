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
	"./conf"
)

var VERSION = "v2.0.3(180310)"

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

