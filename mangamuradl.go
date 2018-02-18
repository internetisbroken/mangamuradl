// mangamura downloader
// 2018/02/17 最初の動くもの
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
	"regexp"
	"strings"
	"strconv"
	"math/rand"
	"sync"
)

var VERSION = "v1.0(180217)"

func Setup() (err error) {
	rand.Seed(time.Now().UnixNano())
	http.DefaultClient.Timeout, _ = time.ParseDuration("20s")
	http.DefaultClient.Transport = http.DefaultTransport
	http.DefaultTransport.(*http.Transport).MaxIdleConns = 1
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 1

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


func _mmPages2enginioLines(content string) (lines []string) {
	re := regexp.MustCompile(`"(\d+)"\s*:\s*{[^}]*"img"\s*:\s*"(.*?)".*?}`)
	matched := re.FindAllStringSubmatch(content, -1)

	for i := 0; i < len(matched); i++ {
		num := matched[i][1]
		url := strings.Replace(matched[i][2], `\`, "", -1)

		lin0 := fmt.Sprintf(`42["request_img",{"url":"%s","cookie_data":"test","id":%s,"viewr":"o"}]`, url, num)
		lin1 := fmt.Sprintf("%d:%s", len(lin0), lin0)

		lines = append(lines, lin1)
	}
	return
}

func mmPages(pageId, pagex string) (lines []string, err error) {
	uri := fmt.Sprintf("http://mangamura.org/pages/xge6?id=%s&x=%s&1", pageId, pagex)
	content, err := httpGetText(uri)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	lines = _mmPages2enginioLines(content)

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
	postdata := strings.Join(lines, "")
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

func DownloadImage(root, id, url string) (err error) {
	JPG := ".jpg"

	testpostfix := []string{JPG, ""}
	for i := 0; i < len(testpostfix); i++ {
		filename := fmt.Sprintf("%s/%s%s", root, id, testpostfix[i])
	///fmt.Printf("testing: %s\n", filename)
		_, e := os.Stat(filename)
		if e == nil || os.IsExist(e) {
			fmt.Printf("Skip: %s\n", filename)
			return;
		}
	}

	content, err := httpGetByte(url)
	if err != nil {
		return
	}

	var postfix string
	if 0xff == content[0] && 0xd8 == content[1] {
		postfix = JPG
	} else {
		fmt.Printf("Unknown format. id: %s, ", id)
		fmt.Printf("header is %x %x %x %x\n", content[0], content[1], content[2], content[3])
	}

	filename := fmt.Sprintf("%s/%s%s", root, id, postfix)
	file, err := os.Create(filename)
	if err != nil {
		return
	}
	defer file.Close()

	_, err = file.Write(content)

	return
}

func opt_pageid() (pageId string, err error) {
	args := os.Args[1:]
	re_0 := regexp.MustCompile(`p=(\w+)`)
	for i := 0; i < len(args); i++ {
		m_0 := re_0.FindStringSubmatch(args[i])
		if len(m_0) >= 2 {
			pageId = m_0[1]
			return
		}
	}
	if pageId == "" {
		re_1 := regexp.MustCompile(`(\d{9,})`)
		for i := 0; i < len(args); i++ {
			m_1 := re_1.FindStringSubmatch(args[i])
			if len(m_1) >= 2 {
				pageId = m_1[1]
				return
			}
		}
	}
	if pageId == "" {
		err = errors.New("page id required");
		return
	}
	return
}

func main() {
	fmt.Printf("version %s\n", VERSION)

	pageId, err := opt_pageid()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	Setup()

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

	pagelist, err := mmPages(pageId, pagex)
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

			fmt.Printf("%s => %s\n", id, url)
			DownloadImage(root, id, url)

			wait.Done()
		}(title, imageInfo[i][0], imageInfo[i][1])
		if i % 16 == 15 {
			wait.Wait()
		}
	}
	wait.Wait()

	fmt.Printf("Done: %s\n", title);
	return
}
