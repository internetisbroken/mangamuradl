// 180302 created

package mmdl

import (
	"github.com/sclevine/agouti"
	"time"
	"fmt"
	"errors"
	"strconv"
	"io/ioutil"
	"database/sql"
	"../conf"
)
//	"./conf"

func openDriver() (driver *agouti.WebDriver, err error) {
	opt := agouti.Timeout(3)
	driver = agouti.ChromeDriver(opt)
	err = driver.Start()
	return
}

func openPage(pageid string, driver *agouti.WebDriver) (page *agouti.Page, err error) {
	page, err = driver.NewPage()
	if err != nil {
		return
	}
	err = page.SetImplicitWait(1000); if err != nil { return }
	err = page.SetPageLoad(10000); if err != nil { return }
	err = page.SetScriptTimeout(10000); if err != nil { return }
	uri := fmt.Sprintf("http://mangamura.org/?p=%s", pageid)
	err = page.Navigate(uri)

	return
}

func runjs(page *agouti.Page, script string) (res interface{}, err error) {
	var arg map[string] interface{}
	err = page.RunScript(script, arg, &res)
	return
}

func loadjs(js string, page *agouti.Page) (res interface{}, err error) {
	content, err := ioutil.ReadFile(js)
	if err != nil {
		return
	}

	res, err = runjs(page, string(content))
	return
}

func getPageInfo(pageid string, tx *sql.Tx, page *agouti.Page) (title string, err error) {

	var res interface{}

	// title
	res, err = runjs(page, `return mmdl.get_title()`)
	if err != nil {
		return
	}
	switch res.(type) {
		case string:
			title = res.(string)
		default:
			err = errors.New("title not found")
			return
	}
	if title == "" {
		err = errors.New("title not found")
		return
	}

	_, err = tx.Exec("insert or ignore into pageinfo(title) values ($1)", title)
	if err != nil {
		return
	}

	// page list
	stmt, err := tx.Prepare("insert or ignore into page(pagenum, path) values(?, ?)")
	if err != nil {
		return
	}
	defer stmt.Close()

	res, err = runjs(page, fmt.Sprintf(`return mmdl.get_list(%s)`, pageid))
	if err != nil {
		return
	}

	var data_i map[string]interface {}
	switch res.(type) {
		case map[string]interface {}:
			data_i = res.(map[string]interface {})
		default:
			fmt.Printf("res:\n%s\n", res)
			err = errors.New("page list not found")
			return
	}

	for id, v := range data_i {
		var v_i map[string]interface {}
		switch v.(type) {
			case map[string]interface {}:
				v_i = v.(map[string]interface {})
			default:
				err = fmt.Errorf("data is not map[string]interface {}\n")
				return
		}
		var url string
		switch v_i["img"].(type) {
			case string:
				url = v_i["img"].(string)
			default:
				err = fmt.Errorf("img is not string\n")
				return
		}
		pagenum, e := strconv.Atoi(id)
		if e != nil {
			err = e
			return
		}
		// db
		_, err = stmt.Exec(pagenum, url)
		if err != nil {
			return
		}
		//fmt.Printf("%d %s\n", pagenum, url)
	}

	return
}

func getImageUrl(db *sql.DB, page *agouti.Page) (err error) {

	var res interface{}

	cookie, _ := conf.GetCookie()

	_, err = runjs(page, fmt.Sprintf(`mmdl.get_prepared(); mmdl.cookie_update("%s");`, cookie))
	if err != nil {
		return
	}

	rows, err := db.Query("select path, pagenum from page where req_status != 2 order by pagenum")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var path string
		var pagenum int
		err = rows.Scan(&path, &pagenum)
		if err != nil {
			return
		}

		_, err = runjs(page, fmt.Sprintf(`return mmdl.sio_request_add(%d, "%s")`, pagenum, path))
		if err != nil {
			return
		}

	}
	err = rows.Err()
	if err != nil {
		return
	}


	stmt, err := db.Prepare("update page set req_status = 2, url = ?, is_frame = ?, is_blob = ?, blob_b64 = ? where pagenum = ?")
	if err != nil {
		return
	}
	defer stmt.Close()

	// function
	insert := func(key string, arr []interface {}) (inserted int, err error) {
		var is_img bool
		var is_iframe bool
		var is_blob bool
		switch key {
			case "img": is_img = true
			case "iframe": is_iframe = true
			case "blob": is_blob = true
			default:
				err = fmt.Errorf("unknown key: %s", key)
				return
		}
		for _, data := range arr {
			switch data.(type) {
				case map[string]interface {}:
					var id int
					var img string
					var b string
					switch data.(map[string]interface {})["id"].(type) {
						case float64:
							id = int(data.(map[string]interface {})["id"].(float64))
						default:
							err = fmt.Errorf("data/id is not float64\n")
							return
					}
					switch data.(map[string]interface {})["img"].(type) {
						case string:
							img = data.(map[string]interface {})["img"].(string)
						default:
							if is_img || is_iframe {
								err = fmt.Errorf("data/img is not string\n")
								return
							}
					}
					switch data.(map[string]interface {})["b"].(type) {
						case string:
							b = data.(map[string]interface {})["b"].(string)
						default:
							if is_blob {
								err = fmt.Errorf("data/b is not string\n")
								return
							}
					}

					_, err = stmt.Exec(img, is_iframe, is_blob, b, id)
					if err != nil {
						return
					}

					inserted++
				default:
					err = fmt.Errorf("data is not map[string]interface {}\n")
					return
			}
		} // for array

		return
	} // end function

	var ntries int
	for {
		res, err = runjs(page, fmt.Sprintf(`return mmdl.sio_do(%d)`, 24))
		if err != nil {
			return
		}
		//fmt.Printf("%v\n", res)

		var res_s map[string]interface {}
		var n_request int
		var n_pending int
		var code int
		switch res.(type) {
			case map[string]interface {}:
				res_s = res.(map[string]interface {})
			default:
				err = fmt.Errorf("data is not map[string]interface {}\n")
				return
		}

		switch res_s["code"].(type) {
			case float64:
				code = int(res_s["code"].(float64))
			default:
				err = fmt.Errorf("code is not float64\n")
				return
		}
		switch res_s["n_request"].(type) {
			case float64:
				n_request = int(res_s["n_request"].(float64))
			default:
				err = fmt.Errorf("n_request is not float64\n")
				return
		}
		switch res_s["n_pending"].(type) {
			case float64:
				n_pending = int(res_s["n_pending"].(float64))
			default:
				err = fmt.Errorf("n_pending is not float64\n")
				return
		}

		var ins int
		var inserted int
		// img
		switch res_s["img"].(type) {
			case []interface {}:
				ins, err = insert("img", res_s["img"].([]interface {}))
				if err != nil {
					return
				}
				inserted += ins
			default:
				err = fmt.Errorf("img is not array\n")
				return
		}

		// iframe
		switch res_s["iframe"].(type) {
			case []interface {}:
				ins, err = insert("iframe", res_s["iframe"].([]interface {}))
				if err != nil {
					return
				}
				inserted += ins
			default:
				err = fmt.Errorf("iframe is not array\n")
				return
		}

		// blob
		switch res_s["blob"].(type) {
			case []interface {}:
				ins, err = insert("blob", res_s["blob"].([]interface {}))
				if err != nil {
					return
				}
				inserted += ins
			default:
				err = fmt.Errorf("blob is not array\n")
				return
		}


		if n_request == 0 && n_pending == 0 {
			break // done!
		} else if code != -2 {
			// not reCapturing
			if inserted == 0 {
				ntries++
			} else {
				ntries = 0
			}
		}

		if ntries >= 10 {
			break
		}

		time.Sleep(1 * time.Second)
	}

	// save cookie
	res, err = runjs(page, `mmdl.sio_disconnect(); return mmdl.cookie_get()`)
	if err != nil {
		return
	}
	switch res.(type) {
		case string:
			cookie = res.(string)
		default:
			err = fmt.Errorf("cookie_get() is not string\n")
			return
	}

	//fmt.Printf("%s\n", cookie)
	err = conf.SetCookie(cookie)
	if err != nil {
		return
	}

	return
}

func Mmdl(pageId string, db *sql.DB) (title string, err error) {

	driver, err := openDriver()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	defer driver.Stop()

	page, err := openPage(pageId, driver)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	_, err = loadjs("js/mmdl.js", page)
	if err != nil {
		return
	}

	tx, err := db.Begin()
	if err != nil {
		return
	}
	// dot org
	title, err = getPageInfo(pageId, tx, page)
	if err != nil {
		return
	}
	tx.Commit()


	// socket.io
	err = getImageUrl(db, page)
	if err != nil {
		return
	}

	return
}
