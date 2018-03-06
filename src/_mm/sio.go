// 180228 created

package mm

import (
	"fmt"
	"regexp"
	"strings"
	"errors"
	"encoding/json"
	"log"
	"os"
	"time"
	"database/sql"
	"../conf"
	"../eio"
	"../httpwrap"
)

var headers = map[string] string{
"Connection": "keep-alive",
"Cache-Control": "no-cache",
"Pragma": "no-cache",
"Accept": "*/*",
"Origin": "http://mangamura.org",
"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.186 Safari/537.36",
"DNT": "1",
"Accept-Encoding": "gzip, deflate",
"Accept-Language": "ja-JP,ja;q=0.9,en-US;q=0.8,en;q=0.7",
}

var headers2 = map[string] string{
"Connection": "keep-alive",
"Cache-Control": "no-cache",
"Pragma": "no-cache",
"Accept": "*/*",
"Origin": "http://mangamura.org",
"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.186 Safari/537.36",
"Content-type": "text/plain;charset=UTF-8",
"DNT": "1",
"Accept-Encoding": "gzip, deflate",
"Accept-Language": "ja-JP,ja;q=0.9,en-US;q=0.8,en;q=0.7",
}

func SioConnect(domain string) (sid string, err error) {
	uri := sioUriFirst(domain)
	content, err := httpwrap.HttpGetTextH(uri, headers)
	if err != nil {
		fmt.Printf("error in SioConnect\n")
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

	uri = sioUri(domain, sid)

	cookie, _ := conf.GetCookie()

	var lin0 string
	if true {
		lin0 = fmt.Sprintf(`42["first_connect","%s"]`, cookie)
	} else {
		lin0 = fmt.Sprintf(`42["first_connect",""]`)
	}
	postdata := fmt.Sprintf("%d:%s", len(lin0), lin0)
//fmt.Printf("post>>%s<<post\n", postdata)

	content, err = httpwrap.HttpPostTextH(uri, postdata, headers2)
	if err != nil {
		return
	}

	return
}

func SioRequest0(domain, sid string, tx *sql.Tx, req_max int) (req_num int, err error) {
	// index: pagenum/req_status
	rows, err := tx.Query("select path, pagenum from page where req_status != 2 order by pagenum limit ?", req_max)
	if err != nil {
		return
	}
	defer rows.Close()

	// index: pagenum
	stmt, err := tx.Prepare("update page set req_status = 1 where pagenum = ?")
	if err != nil {
		return
	}
	defer stmt.Close()

	cookie, _ := conf.GetCookie()

	var postdata string
	for rows.Next() {
		var path string
		var pagenum int
		err = rows.Scan(&path, &pagenum)
		if err != nil {
			return
		}

		lin0 := fmt.Sprintf(`42["request_img",{"url":"%s","cookie_data":"%s","id":%d,"viewr":"o"}]`, path, cookie, pagenum)
		lin1 := fmt.Sprintf("%d:%s", len(lin0), lin0)
		postdata += lin1
		req_num++

		_, err = stmt.Exec(pagenum)
		if err != nil {
			return
		}
	}
	err = rows.Err()
	if err != nil {
		return
	}

	//fmt.Printf("post>>%s<<post\n", postdata)

	if req_num >= 1 {
		uri := sioUri(domain, sid)
		content, e := httpwrap.HttpPostTextH(uri, postdata, headers2)
		if e != nil {
			err = e
			return
		}

		if ! strings.Contains(content, "ok") {
			fmt.Printf("content: %s", content)
			err = errors.New("Post request_img failured")
			return
		}
	}

	return
}

func SioRequest1(domain, sid string, tx *sql.Tx) (resp_num int, disconnecting, require_auth, cookie_delete, cookie_update bool, new_cookie string, err error) {

	stmt, err := tx.Prepare("update page set req_status = 2, url = ?, is_frame = ?, is_blob = ?, blob_b64 = ? where pagenum = ?")
	if err != nil {
		return
	}
	defer stmt.Close()

	for i := 0; i < 2 ; i++ {

		row := tx.QueryRow("select count(id) from page where req_status = 1")
		var requesting int
		err = row.Scan(&requesting)
		if err != nil {
			log.Fatal(err)
		}
		if requesting <= 0 {
			break
		}


		if i >= 1 {
			time.Sleep(500 * time.Millisecond)
			//fmt.Println("test ping")
			uri := sioUri(domain, sid)
			content, e := httpwrap.HttpPostTextH(uri, `1:2`, headers2)
			if e != nil {
				err = e
				return
			}
			if ! strings.Contains(content, "ok") {
				err = errors.New("Post ping failured")
				return
			}
		}

		uri := sioUri(domain, sid)
		content, e := httpwrap.HttpGetTextH(uri, headers)
		if e != nil {
			err = e
			return
		}
		fmt.Printf("get response>>%s<<get response\n", content)

		bytes := []byte(content)
		pos := 0

		for {
			var eio_num int
			var sio_num int
			var payload []byte
			finish, e := eio.Parse(bytes, &pos, &eio_num, &sio_num, &payload)
			if e != nil {
				err = e
				return
			}
			//fmt.Printf("%v %v\n", eio_num, sio_num)
			if eio_num == 4 && sio_num == 2 {
				var data interface{}
				err = json.Unmarshal(payload, &data)
				if err != nil {
					fmt.Printf("json input: %s\n", string(content))
					return
				}

				var data_arr []interface {}
				switch data.(type) {
					case []interface {}:
						data_arr = data.([]interface {})
					default:
						log.Fatalf(`data is not []interface {}\n%v\n`, data)
				}

				var message string
				switch data_arr[0].(type) {
					case string:
						message = data_arr[0].(string)
					default:
						log.Fatalf(`arr[0] is not string\n%v\n`, data)
				}

				if message == "return_img" || message == "return_iframe" {

					var data_i map[string]interface {}
					switch data_arr[1].(type) {
						case map[string]interface {}:
							data_i = data_arr[1].(map[string]interface {})
						default:
							log.Fatalf(`arr[1] is not map[string]interface {}\n%v\n`, data)
					}

					var id int
					var img_url string
					switch data_i["id"].(type) {
						case float64:
							id_f := data_i["id"].(float64)
							id = int(id_f)
						default:
							log.Fatalf(`"id" is not float64\n%v\n`, data)
					}

					switch data_i["img"].(type) {
						case string:
							img_url = data_i["img"].(string)
						default:
							log.Fatalf(`"img" is not string\n%v\n`, data)
					}

					is_frame := (message == "return_iframe")

					fmt.Printf("%d: %s\n", id, img_url)

					_, err = stmt.Exec(img_url, is_frame, 0, "", id)
					if err != nil {
						return
					}
					resp_num++

				} else if message == "return_blob" {

					var data_i map[string]interface {}
					switch data_arr[1].(type) {
						case map[string]interface {}:
							data_i = data_arr[1].(map[string]interface {})
						default:
							log.Fatalf(`arr[1] is not map[string]interface {}\n%v\n`, data)
					}

					var id int
					var blob_b64 string
					var img_url string

					switch data_i["id"].(type) {
						case float64:
							id_f := data_i["id"].(float64)
							id = int(id_f)
						default:
							log.Fatalf(`"id" is not float64\n%v\n`, data)
					}

					switch data_i["b"].(type) {
						case string:
							blob_b64 = data_i["b"].(string)
						default:
							log.Fatalf(`"b" is not string\n%v\n`, data)
					}

					switch data_i["img"].(type) {
						case string:
							img_url = data_i["img"].(string)
						default:
							;
					}

					fmt.Printf("%d: blob\n", id)

					_, err = stmt.Exec(img_url, 0, 1, blob_b64, id)
					if err != nil {
						log.Fatal(err)
					}
					resp_num++

				} else if message == "require_auth" {
					require_auth = true

				} else if message == "cookie_delete" {
					cookie_delete = true

				} else if message == "cookie_update" {
					cookie_update = true
					switch data_arr[1].(type) {
						case string:
							new_cookie = data_arr[1].(string)
						default:
							log.Fatalf(`arr[1] is not string\n%v\n`, data)
					}

				} else {
					log.Fatalf(`[FIXME] message: %s\n%v\n`, data)
				}

			} else if eio_num == 4 && sio_num == 1 {
				disconnecting = true

			} else if eio_num == 3 {
				//fmt.Println("pong received")

			} else {
				fmt.Printf("[FIXME] eio_num %v, sio_num %v\n", eio_num, sio_num)
				os.Exit(1)
			}

			if finish {
				break
			}
		} // for parse eio

		if disconnecting {
			break
		}

		if resp_num <= 0 {
			fmt.Println("No message")
			break
		}
	} // for 1..2

	return
}

func sioUriFirst(domain string) (uri string) {
	t := eio.EncodedTime()
	uri = fmt.Sprintf("http://%s/socket.io/?EIO=3&transport=polling&t=%s", domain, t)

	return
}
func sioUri(domain, sid string) (uri string) {
	t := eio.EncodedTime()
	uri = fmt.Sprintf("http://%s/socket.io/?EIO=3&transport=polling&t=%s&sid=%s", domain, t, sid)

	return
}
