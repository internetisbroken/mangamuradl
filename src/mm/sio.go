// 180228 created

package mm

import (
	"fmt"
	"regexp"
	"strings"
	"errors"
	"encoding/json"
	"log"
	"reflect"
	"os"
	"time"
	"database/sql"
	"../conf"
	"../eio"
	"../httpwrap"
)

func SioConnect(domain string) (sid string, err error) {
	uri := sioUriFirst(domain)
	content, err := httpwrap.HttpGetText(uri)
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

	content, err = httpwrap.HttpPostText(uri, postdata)
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
		content, e := httpwrap.HttpPostText(uri, postdata)
		if e != nil {
			err = e
			return
		}

		if ! strings.Contains(content, "ok") {
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
			content, e := httpwrap.HttpPostText(uri, `1:2`)
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
		content, e := httpwrap.HttpGetText(uri)
		if e != nil {
			err = e
			return
		}
		//fmt.Printf("get response>>%s<<get response\n", content)

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
				if reflect.TypeOf(data).String() == "[]interface {}" {
					data_arr = data.([]interface {})
				} else {
					fmt.Printf("Unknown data type: %v\n", reflect.TypeOf(data))
					fmt.Printf("%v\n", data)
					log.Fatal("error")
				}

				var message string
				if reflect.TypeOf(data_arr[0]).String() == "string" {
					message = data_arr[0].(string)
				} else {
					fmt.Printf("Unknown data type: %v\n", reflect.TypeOf(data_arr[0]))
					fmt.Printf("%v\n", data_arr[0])
					log.Fatal("error")
				}

				if message == "return_img" || message == "return_iframe" {

					var data_i map[string]interface {}
					if reflect.TypeOf(data_arr[1]).String() ==  "map[string]interface {}" {
						data_i = data_arr[1].(map[string]interface {})
					} else {
						fmt.Printf("Unknown data type: %v\n", reflect.TypeOf(data_arr[1]))
						fmt.Printf("%v\n", data_arr[1])
						log.Fatal("error")
					}

					var id int
					var img_url string

					if reflect.TypeOf(data_i["id"]).String() == "float64" {
						id_f := data_i["id"].(float64)
						id = int(id_f)
					} else {
						log.Fatal("reflect.TypeOf(data_i[id]) != float64")
					}

					if reflect.TypeOf(data_i["img"]).String() == "string" {
						img_url = data_i["img"].(string)
					} else {
						log.Fatal("reflect.TypeOf(data_i[img]) != string")
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
					if reflect.TypeOf(data_arr[1]).String() ==  "map[string]interface {}" {
						data_i = data_arr[1].(map[string]interface {})
					} else {
						fmt.Printf("Unknown data type: %v\n", reflect.TypeOf(data_arr[1]))
						fmt.Printf("%v\n", data_arr[1])
						log.Fatal("error")
					}

					var id int
					var blob_b64 string
					var img_url string

					if reflect.TypeOf(data_i["id"]).String() == "float64" {
						id_f := data_i["id"].(float64)
						id = int(id_f)
					} else {
						log.Fatal("reflect.TypeOf(data_i[id]) != float64")
					}

					if reflect.TypeOf(data_i["b"]).String() == "string" {
						blob_b64 = data_i["b"].(string)
					} else {
						log.Fatal("reflect.TypeOf(data_i[id]) != float64")
					}

					if reflect.TypeOf(data_i["img"]).String() == "string" {
						img_url = data_i["img"].(string)
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
					if reflect.TypeOf(data_arr[1]).String() == "string" {
						new_cookie = data_arr[1].(string)
					} else {
						log.Fatalf("data of cookie_update is not string, %v\n", reflect.TypeOf(data_arr[1]))
					}

				} else {
					fmt.Printf("[FIXME] message: %s, %v\n", message, data_arr[1])
					fmt.Printf("%s\n", reflect.TypeOf(data_arr[1]).String() == "string")
					os.Exit(1)
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
