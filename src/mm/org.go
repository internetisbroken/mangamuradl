// 180228 created

package mm

import (
	"fmt"
	"regexp"
	"errors"
	"strconv"
	"math/rand"
	"net/http"
	"net/url"
	"encoding/json"
	"database/sql"
	"../httpwrap"
)

func GetTitle(pageId string) (title string, err error) {
	uri := fmt.Sprintf("http://mangamura.org/?p=%s", pageId)
	content, err := httpwrap.HttpGetText(uri)

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

func GetSocketioServer(page string) (server string, err error) {
	uri := fmt.Sprintf("http://mangamura.org/kai_pc_viewer?p=%s", page)
	content, err := httpwrap.HttpGetText(uri)
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
		if sub == 0 {
			sub = 1
		}
		server = fmt.Sprintf("socket%d.spimg.ch", sub)
	} else {
		fmt.Println(content)
		err = errors.New("Socket.io server not decided")
	}

	return
}

func getPageX(pageId string) (pagex string, err error) {
	uri := fmt.Sprintf("http://mangamura.org/pages/getxb?wp_id=%s", pageId)

	content, err := httpwrap.HttpGetText(uri)
	if err != nil {
		return
	}
	pagex = content

	return
}

func GetPageList(pageId string, db *sql.DB) (err error) {
	pagex, err := getPageX(pageId)
	if err != nil {
		fmt.Printf("getPageX: %v\n", err)
		return
	}
	//fmt.Printf("pagex is %v\n", pagex)

	uri := fmt.Sprintf("http://mangamura.org/pages/xge6?id=%s&x=%s&1", pageId, pagex)
	content, err := httpwrap.HttpGetByte(uri)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	//fmt.Printf("%v\n", string(content))

	var data interface{}

	err = json.Unmarshal(content, &data)
	if err != nil {
		fmt.Printf("json input: %s\n", string(content))
		return
	}

	// start transaction
	tx, err := db.Begin()
	if err != nil {
		return
	}
	stmt, err := tx.Prepare("insert or ignore into page(path, pagenum) values(?, ?)")
	if err != nil {
		return
	}
	defer stmt.Close()

	for k, v := range data.(map[string]interface{}) {
		img_i := v.(map[string]interface{})["img"]
		if img_i != nil {
			num, e := strconv.Atoi(k)
			if e != nil {
				err = e
				return
			}
			img := img_i.(string)

			_, err = stmt.Exec(img, num)
			if err != nil {
				return
			}
		}
	}
	tx.Commit()

	return
}


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
