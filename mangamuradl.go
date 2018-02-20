// mangamura downloader
//
// 上げたらvar VERSIONを更新すること
//
// 2018/02/17 最初の動くもの
// v1.0.1(180219) -cookie を追加(reCapture)
// v1.0.2(180220) getcookie に対応

package main

import (
	"fmt"
	"time"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"golang.org/x/net/publicsuffix"
	"io"
	"io/ioutil"
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"strconv"
	"math/rand"
	"sync"
	"github.com/go-ini/ini"
)

var VERSION = "v1.0.2(180220)"

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

func _httpReq(req *http.Request) (content []byte, err error) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/63.0.3239.132 Safari/537.36")
	req.Header.Set("Origin", "http://mangamura.org");

	//---req.Close = true
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	content, err = ioutil.ReadAll(res.Body)
	io.Copy(ioutil.Discard, res.Body)
	res.Body.Close()
	return
}

func httpPostByte(url string, data string) (content []byte, err error) {
	reader := strings.NewReader(data)
	req, err := http.NewRequest("POST", url, reader);
	if err != nil {
		return
	}

	return _httpReq(req)
}
func httpPostText(url string, data string) (content string, err error) {
	bytes, err := httpPostByte(url, data)
	if err != nil {
		return
	}
	content = string(bytes)
	return
}


func httpGetByte(url string) (content []byte, err error) {
	req, err := http.NewRequest("GET", url, nil);
	if err != nil {
		return
	}

	return _httpReq(req)
}

func httpGetText(url string) (content string, err error) {
	bytes, err := httpGetByte(url)
	if err != nil {
		return
	}
	content = string(bytes)
	return
}

// yeast
func sioEncodedTime() (encoded string) {
	// config
	str := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-_"
	var length int64
	length = 64

	var num int64
	num = time.Now().UnixNano()/(1000*1000)

	alphas := strings.Split(str, "")

	for {
		encoded = alphas[num % length] + encoded

		num = num / length
		if num <= 0 {
			break
		}
	}

	return
}

//
func setAuthview(content string) (err error) {
	re := regexp.MustCompile(`"authview"\s*,\s*"([^"]+)"`)
	matched := re.FindStringSubmatch(content)

	if len(matched) >=2 {
		uri, _ := url.Parse("http://mangamura.org/")
		var cookies []*http.Cookie

		cookie := &http.Cookie{
			Name: "authview",
			Value: matched[1],
			Path: "/",
			Domain: "mangamura.org",
		}
		cookies = append(cookies, cookie)

		http.DefaultClient.Jar.SetCookies(uri, cookies)
	} else {
		err = errors.New("authview not found")
	}
	return
}


func mmSocketioServer(page string) (server string, err error) {
	uri := fmt.Sprintf("http://mangamura.org/kai_pc_viewer?p=%s", page)
	content, err := httpGetText(uri)
	if err != nil {
		return
	}

	re0 := regexp.MustCompile(`(socket\d+\.spimg\.ch)`)
	ma0 := re0.FindStringSubmatch(content)
	if len(ma0) >=2 {
		server = ma0[1]
		return
	}

	re := regexp.MustCompile(`\Dmax\s*=\s*(\d+)\s*;`)

	matched := re.FindStringSubmatch(content)
	if len(matched) >=2 {
		max, e := strconv.Atoi(matched[1])
		if e != nil {
			err = e
			return
		}

		sub := rand.Intn(max)
		server = fmt.Sprintf("socket%d.spimg.ch", sub)
	} else {
		err = errors.New("Socket.io server not decided")
	}

	return
}

func mmTitle(pageId string) (title string, err error) {
	uri := fmt.Sprintf("http://mangamura.org/?p=%s", pageId)
	content, err := httpGetText(uri)

	err = setAuthview(content)
	if err != nil {
		return
	}

	re := regexp.MustCompile(`<li class="title"><h2 class="normalfont">(.+?)\s*<`)
	matched := re.FindStringSubmatch(content)

	if len(matched) >=2 {
		title = matched[1]
	} else {
		err = errors.New("Title Not Found")
	}

	return
}


func mmPageX(pageId string) (pagex string, err error) {
	uri := fmt.Sprintf("http://mangamura.org/pages/getxb?wp_id=%s", pageId)

	content, err := httpGetText(uri)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	pagex = content

	return
}


func _mmPages2enginioLines(content, cookie string) (lines []string) {
	re := regexp.MustCompile(`"(\d+)"\s*:\s*{[^}]*"img"\s*:\s*"(.*?)".*?}`)
	matched := re.FindAllStringSubmatch(content, -1)

	for i := 0; i < len(matched); i++ {
		num := matched[i][1]
		url := strings.Replace(matched[i][2], `\`, "", -1)
		// "cookie_data":"test"
		lin0 := fmt.Sprintf(`42["request_img",{"url":"%s","cookie_data":"%s","id":%s,"viewr":"o"}]`, url, cookie, num)
		lin1 := fmt.Sprintf("%d:%s", len(lin0), lin0)

		lines = append(lines, lin1)
	}
	return
}

func mmPages(pageId, pagex, cookie string) (lines []string, err error) {
	uri := fmt.Sprintf("http://mangamura.org/pages/xge6?id=%s&x=%s&1", pageId, pagex)
	content, err := httpGetText(uri)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	lines = _mmPages2enginioLines(content, cookie)

	return
}


func _mmsioUriFirst(domain string) (uri string) {
	t := sioEncodedTime()
	uri = fmt.Sprintf("http://%s/socket.io/?EIO=3&transport=polling&t=%s", domain, t)

	return
}
func _mmsioUri(domain, sid string) (uri string) {
	t := sioEncodedTime()
	uri = fmt.Sprintf("http://%s/socket.io/?EIO=3&transport=polling&t=%s&sid=%s", domain, t, sid)
	return
}

func mmsioConnect(domain string) (sid string, err error) {
	uri := _mmsioUriFirst(domain)
	content, err := httpGetText(uri)
	if err != nil {
		return
	}

	//fmt.Printf("%s\n", content)

	re := regexp.MustCompile(`"sid"\s*:\s*"(.+?)"`)
	matched := re.FindStringSubmatch(content)

	if len(matched) >=2 {
		sid = matched[1]
	} else {
		err = errors.New("sid Not Found")
	}

	return
}

func mmsioRequest0(domain, sid string, lines []string) (err error) {
	uri := _mmsioUri(domain, sid)
	postdata := strings.Join(lines[:], "")
	// fmt.Printf("post>>%s<<post", postdata)
	content, err := httpPostText(uri, postdata)
	if err != nil {
		return
	}
	// fmt.Printf("post response>>%s<<post response", content)

	if ! strings.Contains(content, "ok") {
		err = errors.New("Post request_img failured")
		return
	}

	return
}

func _mmImageInfo(content string) (imageInfo [][]string) {
	re := regexp.MustCompile(`:42\[.*?\]`)
	matched := re.FindAllStringSubmatch(content, -1)

	re_img := regexp.MustCompile(`"img"\s*:\s*"(.*?)"`)
	re_id := regexp.MustCompile(`"id":(\d+)`)

	for i := 0; i < len(matched); i++ {
		m_img := re_img.FindStringSubmatch(matched[i][0])
		m_id := re_id.FindStringSubmatch(matched[i][0])
		if len(m_img) >= 2 && len(m_id) >=2 {
			var line [2]string
			line[0] = m_id[1]
			line[1] = m_img[1]
			imageInfo = append(imageInfo, line[:])
			//fmt.Printf("%s %s\n", m_id[1], m_img[1])
		}
	}
	return
}

// return: imageinfo[i][0]: image id(1,2,...), imageinfo[i][1]: url
func mmsioRequest1(domain, sid string) (imageInfo [][]string, err error) {
	uri := _mmsioUri(domain, sid)
	content, err := httpGetText(uri)
	if err != nil {
		return
	}
	//fmt.Printf("get response>>%s<<get response", content)

	imageInfo = _mmImageInfo(content)
	return
}

func FindImageByNumber(root, num string) (exist bool, filename string) {
	JPG := ".jpg"

	testpostfix := []string{JPG, ""}
	fileformat := []string{"%s/%s%s", "%s/0%s%s", "%s/00%s%s", "%s/000%s%s"}
	for i := 0; i < len(testpostfix); i++ {
		for j := 0; j < len(fileformat); j++ {
			filename = fmt.Sprintf(fileformat[j], root, num, testpostfix[i])
			//fmt.Printf("testing: %s\n", filename)
			_, e := os.Stat(filename)
			if e == nil || os.IsExist(e) {
				exist = true
				return;
			}
		}
	}
	exist = false
	return
}


func DownloadImage(root, id, url string) (err error) {
	JPG := ".jpg"

	exists, testname := FindImageByNumber(root, id)
	if exists {
		fmt.Printf("Skip: %s\n", testname)
		return
	}

	content, err := httpGetByte(url)
	if len(content) < 10 {
		err = errors.New("Download failured: " + url)
		return
	}

	if err != nil {
		return
	}

	var postfix string
	if 0xff == content[0] && 0xd8 == content[1] {
		postfix = JPG
	} else {
		errstr := fmt.Sprintf("Unknown format. id: %s, header is %x %x %x %x\n",
			id, content[0], content[1], content[2], content[3])
		err = errors.New(errstr)
		return
	}

	filename := fmt.Sprintf("%s/%s%s", root, id, postfix)
	file, err := os.Create(filename)
	if err != nil {
		return
	}
	defer file.Close()

	_, err = file.Write(content)
	if err != nil {
		return
	}
	fmt.Printf("Saved: %s\n", filename)

	return
}

func get_settings() (pageId, cookie string, err error) {
	args := os.Args[1:]
	re_cookie := regexp.MustCompile(`(?i)^--?cookie=(\S+)`)
	re_0 := regexp.MustCompile(`p=(\w+)`)
	re_1 := regexp.MustCompile(`(\d{9,})`)
	for i := 0; i < len(args); i++ {
		m_cookie := re_cookie.FindStringSubmatch(args[i])
		if len(m_cookie) >= 2 {
			cookie = m_cookie[1]
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

	if cookie == "" {
		c, e := GetIni("mangamuradl.ini", "acookie4")
		if c != "" && e == nil {
			fmt.Println("Cookie loaded")
			cookie = c
		}
	}

	return
}



func UpdateIni(filename, key, val string) (err error) {
	cfg, err := ini.LooseLoad(filename)
	if err != nil {
		return
	}
	cfg.Section("").NewKey(key, val)
	err = cfg.SaveTo(filename)
	return
}

func GetIni(filename, key string) (val string, err error) {
	cfg, err := ini.LooseLoad(filename)
	if err != nil {
		return
	}

	k, err := cfg.Section("").GetKey(key)
	if err != nil {
		return
	}

	val = k.String()
	return
}

func exec_getcookie() {
	fmt.Printf("Chromeが立ち上がるので認証して下さい\n")
	out, err := exec.Command("getcookie").Output()
	fmt.Printf("\n%s\n", string(out))
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("認証できたら、コマンドを再実行して下さい\n")
	}
}

func main() {
	fmt.Printf("version %s\n", VERSION)
	Setup()


	pageId, cookie, err := get_settings()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	title, err := mmTitle(pageId)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	fmt.Printf("title[%v]\n", title)

	pagex, err := mmPageX(pageId)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	//fmt.Printf("pagex is %v\n", pagex)

	pagelist, err := mmPages(pageId, pagex, cookie)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	sioserver, err := mmSocketioServer(pageId)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	//fmt.Printf("sioserver is %v\n", sioserver)

	//<-- Socket.io

	sid, err := mmsioConnect(sioserver)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	err = mmsioRequest0(sioserver, sid, pagelist)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	imageInfo, err := mmsioRequest1(sioserver, sid)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	//<-- download images
	if len(imageInfo) > 0 {
		err := os.Mkdir(title, 0777)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("%v\n", err)
				return
			}
		}
	}

	wait := new(sync.WaitGroup)
	for i := 0; i < len(imageInfo); i++ {
		wait.Add(1)
		go func(root, id, url string) {
			//---fmt.Printf("%s => %s\n", imageInfo[i][0], imageInfo[i][1])
			//---DownloadImage(title, imageInfo[i][0], imageInfo[i][1])

			//fmt.Printf("%s => %s\n", id, url)
			err := DownloadImage(root, id, url)
			if err != nil {
				fmt.Printf("%v\n", err)
			}
			wait.Done()
		}(title, imageInfo[i][0], imageInfo[i][1])
		if i % 16 == 15 {
			wait.Wait()
		}
	}
	wait.Wait()

	if len(pagelist) != len(imageInfo) {
		fmt.Printf("Authentication required\n");
		exec_getcookie()
	}

	fmt.Printf("Done: %s\n", title);
	return
}
