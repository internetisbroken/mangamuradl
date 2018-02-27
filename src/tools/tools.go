// 180228 created

package tools

import (
	"fmt"
	"regexp"
	"errors"
	"os"
	"io"
	"io/ioutil"
	"archive/zip"
	"path/filepath"
	"net/http"
	"../httpwrap"
)

func DownloadChromedriver() (err error) {
	filelist := []string{"chromedriver.exe"}
	msg := "必要なツール(chromedriver)を取得しています"

	ok, _ := testFileList(filelist)
	if ok {
		return
	}

	fmt.Printf("%v\n", msg)

	var url string

	content, err := httpwrap.HttpGetText("https://sites.google.com/a/chromium.org/chromedriver/downloads")
	if err != nil {
		return
	}
	re0 := regexp.MustCompile(`://chromedriver\.storage\.googleapis\.com/index\.html\?path=([^"'><]+?)/`)
	ma0 := re0.FindStringSubmatch(content)
	if len(ma0) >= 2 {
		url = fmt.Sprintf("https://chromedriver.storage.googleapis.com/%s/chromedriver_win32.zip", ma0[1])
	} else {
		err = errors.New("DownloadChromedriver: can't find latest file\n")
		return
	}

	err = downloadZip(url, "chromedriver_win32.zip")
	if err != nil {
		return
	}
	err = extractFiles("chromedriver_win32.zip", filelist)
	if err != nil {
		return
	}

	_, err = testFileList(filelist)

	return
}

func DownloadPhantomJs() (err error) {
	file := []string{"phantomjs.exe"}
	url := "http://phantomjs.org/download.html"
	re := regexp.MustCompile(`(https?://[^\s"'><]+?/(phantomjs-[^/\s"'><]+?windows\.zip))`)
	msg := "必要なツール(PhantomJs)を取得しています"

	err = downloadTool(file, url, re, msg)
	return
}

func DownloadImageMagic() (err error) {
	file := []string{"convert.exe", "magic.xml"}
	url := "https://www.imagemagick.org/script/download.php"
	re := regexp.MustCompile(`(https?://(?:.*?\.)*imagemagick\.org/.*?([^/]*?-portable-Q16-x86\.zip))`)
	msg := "必要なツール(ImageMagic)を取得しています"

	err = downloadTool(file, url, re, msg)
	return
}

func testFileList(filelist []string) (ok bool, err error) {

	var nfound int
	for i := 0; i < len(filelist); i++ {
		_, err = os.Stat(filelist[i])
		if err != nil {
			return
		} else {
			nfound++
		}
	}
	if len(filelist) == nfound {
		ok = true
	}
	return
}

func downloadZip(url, filename string) (err error) {
	out, err := os.Create(filename)
	if err != nil {
		return
	}
	defer out.Close()
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	fmt.Printf("downloading %s\n", url)
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return
	}
	fmt.Printf("saved: %s\n", filename)
	return
}

func extractFiles(filename string, filelist []string) (err error) {
	r, err := zip.OpenReader(filename)
	if err != nil {
		return
	}
	defer r.Close()

	for _, f := range r.File {
		base := filepath.Base(f.Name)
		//fmt.Printf("%v %v\n", base, f.Name)
		var found bool
		for j := 0; j < len(filelist); j++ {
			if base == filelist[j] {
				found = true
			}
		}
		if found {
			rc, e := f.Open();
			if e != nil {
				err = e
				return
			}
			buf := make([]byte, f.UncompressedSize)
			if _, e := io.ReadFull(rc, buf); e != nil {
				err = e
				return
			}

			if e := ioutil.WriteFile(base, buf, f.Mode()); e != nil {
				err = e
				return
			}
		}
	}

	return
}

func downloadTool(filelist []string, url string, re *regexp.Regexp, msg string) (err error) {
	ok, _ := testFileList(filelist)
	if ok {
		return
	}

	fmt.Printf("%v\n", msg)

	content, err := httpwrap.HttpGetText(url)
	if err != nil {
		return
	}

	m := re.FindStringSubmatch(content)
	if len(m) < 3 {
		fmt.Printf("%v\n", m)
		return
	}

	zipurl := m[1]
	zipname := m[2]
	err = downloadZip(zipurl, zipname)
	if err != nil {
		return
	}
	err = extractFiles(zipname, filelist)
	if err != nil {
		return
	}

	_, err = testFileList(filelist)

	return
}
