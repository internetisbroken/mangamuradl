// 180228 created
// 180306 add DownloadUBlock()
// 180315 add 7za
// 180315 add startBrowser

package tools

import (
	"fmt"
	"regexp"
	"errors"
	"os"
	"io"
	"net/http"
	"../httpwrap"
	"time"
	"path"
	"strings"
	"path/filepath"
	"github.com/sclevine/agouti"
)

var toolDir = "tool/"

type Tool struct {
	Dir string
	Exe string
	GetReq func() (*http.Request, string, error)
}

func init() {
	err := os.Mkdir(toolDir, 0777)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("%v\n", err)
		}
	}
}

func getPath(name string) string {
	return path.Clean(fmt.Sprintf("%s/%s", toolDir, name))
}

func StartChrome() (*agouti.WebDriver, *agouti.Page, error) {
	return startBrowser(true)
}
func StartPhantomjs() (*agouti.WebDriver, *agouti.Page, error) {
	return startBrowser(false)
}
func startBrowser(isChrome bool) (driver *agouti.WebDriver, page *agouti.Page, err error) {

	opt := agouti.Timeout(15)

	if isChrome {
		exe, e := GetChromedriver()
		if e != nil {
			err = e
			return
		}

		// extension
		path, e := GetUblock0()
		if e != nil {
			err = e
			return
		}
		dir, e := filepath.Abs(path)
		if e != nil {
			err = e
			return
		}
		// chrome option
		args := []string{
			fmt.Sprintf(`load-extension=%s`, dir),
			"enable-automation",
			//"headless",
		}
		opt_c := agouti.ChromeOptions("args", args)

		cmd := []string{exe, "--port={{.Port}}"}
		driver = agouti.NewWebDriver("http://{{.Address}}", cmd, opt, opt_c)

	} else {
		exe, e := GetPhantomjs()
		if e != nil {
			err = e
			return
		}

		cmd := []string{exe, "--webdriver={{.Address}}", "--ignore-ssl-errors=true"}
		driver = agouti.NewWebDriver("http://{{.Address}}", cmd, opt)
	}

	fmt.Printf("ファイアウォールのメッセージが出る場合、")
	fmt.Printf("キャンセル（不許可）を選んでも問題ありません\n")

	if err = driver.Start(); err != nil {
		return
	}

	page, err = driver.NewPage()
	if err != nil {
		driver.Stop()
		return
	}
	err = page.SetImplicitWait(1000); if err != nil { return }
	err = page.SetPageLoad(15000); if err != nil { return }
	err = page.SetScriptTimeout(15000); if err != nil { return }

	return
}

////////////////////////////////////////////////////////////
// Chromedriver: https://sites.google.com/a/chromium.org/chromedriver/home
////////////////////////////////////////////////////////////
var chromedriver = Tool{
	Dir: "chromedriver",
	Exe: "chromedriver",
	GetReq: reqChromedriver,
}
func GetChromedriver() (exe string, err error) {
	return getBin(chromedriver)
}
func reqChromedriver() (req *http.Request, zipName string, err error) {
	fmt.Printf("必要なツール(chromedriver)を取得しています\n")

	zipName = "chromedriver_win32.zip"
	u0 := "https://sites.google.com/a/chromium.org/chromedriver/downloads"
	content, err := httpwrap.HttpGetText(u0)
	if err != nil {
		return
	}

	re0 := regexp.MustCompile(`://chromedriver\.storage\.googleapis\.com/index\.html\?path=([^"'><]+?)/`)
	ma0 := re0.FindStringSubmatch(content)
	if len(ma0) < 2 {
		err = errors.New("Chromedriver: zip link not found\n")
		return
	}
	url := fmt.Sprintf("https://chromedriver.storage.googleapis.com/%s/chromedriver_win32.zip", ma0[1])
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Referer", u0)

	return
}

////////////////////////////////////////////////////////////
// PhantomJS: http://phantomjs.org/
////////////////////////////////////////////////////////////
var phantomjs = Tool{
	Dir: "phantomjs",
	Exe: "phantomjs",
	GetReq: reqPhantomjs,
}
func GetPhantomjs() (exe string, err error) {
	return getBin(phantomjs)
}
func reqPhantomjs() (req *http.Request, zipName string, err error) {
	fmt.Printf("必要なツール(PhantomJS)を取得しています\n")
	u0 := "http://phantomjs.org/download.html"

	content, err := httpwrap.HttpGetText(u0)
	if err != nil {
		return
	}

	re := regexp.MustCompile(`(https?://[^\s"'><]+?/(phantomjs-[^/\s"'><]+?windows\.zip))`)
	ma := re.FindStringSubmatch(content)
	if len(ma) < 3 {
		err = errors.New("PhantomJS: zip link not found\n")
		return
	}
	url := ma[1]
	zipName = ma[2]
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Referer", u0)

	return
}

////////////////////////////////////////////////////////////
// ImageMagick: https://www.imagemagick.org/script/index.php
////////////////////////////////////////////////////////////
var convert = Tool{
	Dir: "imagemagick",
	Exe: "convert",
	GetReq: reqImageMagick,
}
func GetConvert() (exe string, err error) {
	return getBin(convert)
}
func reqImageMagick() (req *http.Request, zipName string, err error) {
	fmt.Printf("必要なツール(ImageMagick)を取得しています\n")
	u0 := "https://www.imagemagick.org/script/download.php"

	content, err := httpwrap.HttpGetText(u0)
	if err != nil {
		return
	}

	re := regexp.MustCompile(`(https?://(?:.*?\.)*imagemagick\.org/.*?([^/]*?-portable-Q16-x86\.zip))`)
	ma := re.FindStringSubmatch(content)
	if len(ma) < 3 {
		err = errors.New("ImageMagick: zip link not found\n")
		return
	}
	url := ma[1]
	zipName = ma[2]
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Referer", u0)

	return
}


////////////////////////////////////////////////////////////
// 7za: https://www.7-zip.org/
////////////////////////////////////////////////////////////
var sevenza = Tool{
	Dir: "7za",
	Exe: "7za",
	GetReq: req7za,
}
func Get7za() (exe string, err error) {
	return getBin(sevenza)
}
func req7za() (req *http.Request, zipName string, err error) {
	fmt.Printf("必要なツール(7za)を取得しています\n")

	zipName = "7za920.zip"
	u0 := "https://osdn.net/projects/sevenzip/downloads/64455/7za920.zip/"

	content, err := httpwrap.HttpGetText(u0)
	if err != nil {
		return
	}
	re := regexp.MustCompile(`/(frs/redir\.php\?[^"'\s>]+)`)
	m := re.FindStringSubmatch(content)
	if len(m) < 2 {
		err = fmt.Errorf("7za: zip link not found")
		return
	}

	url := fmt.Sprintf(`https://osdn.net/%s`, m[1])
	url = strings.Replace(url,"&amp;", "&", -1)
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Referer", u0)
	return
}

////////////////////////////////////////////////////////////
// uBlock Origin: https://github.com/gorhill/uBlock/
////////////////////////////////////////////////////////////
var ublock0 = Tool{
	Dir: "extension/uBlock0.chromium",
	Exe: "extension/uBlock0.chromium/",
	GetReq: reqUblock0,
}
func GetUblock0() (exe string, err error) {
	return getBin(ublock0)
}
func reqUblock0() (req *http.Request, zipName string, err error) {
	fmt.Printf("Chromeの拡張機能(uBlock Origin)を取得しています\n")

	u0 := "https://github.com/gorhill/uBlock/releases/latest"

	content, err := httpwrap.HttpGetText(u0)
	if err != nil {
		return
	}
	re := regexp.MustCompile(`/(gorhill/uBlock/releases/download/[^'"]+/(uBlock0\.chromium\.zip))`)
	ma := re.FindStringSubmatch(content)

	if len(ma) < 3 {
		err = fmt.Errorf("uBlock Origin: zip link not found")
		return
	}

	url := fmt.Sprintf("https://github.com/%s", ma[1])
	zipName = ma[2]

	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Referer", u0)

	return
}

func findBin(dirName, binName string) (exe string, err error) {
	var testList []string

	dir0 := getPath(dirName)
	fmts0 := []string{`%s/%s`, `%s/%s.exe`, `%s/bin/%s`, `%s/bin/%s.exe`}
	for _, f := range fmts0 {
		testList = append(testList, fmt.Sprintf(f, dir0, binName))
	}

	dir1 := getPath("")
	fmts1 := []string{`%s/%s`, `%s/%s.exe`, `%s/bin/%s`, `%s/bin/%s.exe`}
	for _, f := range fmts1 {
		testList = append(testList, fmt.Sprintf(f, dir1, binName))
	}

	for _, test := range testList {
		//fmt.Printf("debug test bin: %s\n", test)
		if testFile(test) {
			exe = test
			return
		}
	}
	err = fmt.Errorf(`Can't find %s`, binName)
	return
}
// find exe path or download binary
func getBin(tool Tool) (exe string, err error) {
	for i := 0; i <= 1; i++ {
		exe, err = findBin(tool.Dir, tool.Exe);
		if err == nil {
			return
		}
		if i > 0 {
			err = fmt.Errorf("Not found excutable: %s", tool.Exe)
			return
		}

		req, zipName, e := tool.GetReq()
		if e != nil {
			err = e
			return
		}
		if strings.Compare(zipName, "") == 0 {
			err = fmt.Errorf("[FIXME] zip name not specified")
			return
		}
		err = downloadZip(req, zipName)
		if err != nil {
			return
		}
		err = unzip(getPath(zipName), getPath(tool.Dir), true)
		if err != nil {
			return
		}
	}
	return
}

func downloadZip(req *http.Request, filename string) (err error) {

	client := &http.Client{
		Timeout: time.Duration(600) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	path := getPath(filename)
	out, err := os.Create(path)
	if err != nil {
		return
	}
	defer out.Close()

	fmt.Printf("Downloading %v\n", req.URL)
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return
	}
	fmt.Printf("Saved: %s\n", path)
	return
}

func testFile(name string) (bool) {
	_, err := os.Stat(name)
	if err != nil {
		return false
	}
	return true
}
