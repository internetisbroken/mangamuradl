// 180228 created
// 180306 add os.Remove splitted pages
// 180310 remove javascript code

package img

import (
	"fmt"
	"os"
	"os/exec"
	"io/ioutil"
	"strings"
	"errors"
	"encoding/base64"
	"github.com/sclevine/agouti"
	"../httpwrap"
	"../tools"
)

var jpg = ".jpg"

func DownloadImage(root string, pagenum int, url string, is_frame, is_blob bool, base64str string) (err error) {

	exists, testname := findImageByNumber(root, pagenum)
	if exists {
		if testname == "" {
			//fmt.Printf("Skip: %s\n", testname)
		}
		return
	}

	var filename string
	if is_frame {
		filename, err = splittedPage(root, pagenum, url)
		if err != nil {
			return
		}
	} else {
		var content []byte
		if is_blob {
			a := strings.Split(base64str, ",")
			content, err = base64.StdEncoding.DecodeString( a[ len(a) - 1 ] )
			if err != nil {
				return
			}
		} else {
			if url != "" {
				content, err = httpwrap.HttpGetByte(url)
				if err != nil {
					return
				}
			} else {
				err = errors.New("Image url required")
				return
			}
		}

		if len(content) < 10 {
			err = errors.New("Download failured: " + url)
			return
		}

		if 0x3c == content[0] {
			err = errors.New("data is not image: " + url)
			return

		} else {
			var postfix string
			if 0xff == content[0] && 0xd8 == content[1] {
				postfix = jpg
			} else {
				errstr := fmt.Sprintf("Unknown format: %d(%s), header is %x %x %x %x\n",
					pagenum, url, content[0], content[1], content[2], content[3])
				err = errors.New(errstr)
				return
			}

			filename = fmt.Sprintf("%s/%d%s", root, pagenum, postfix)
			file, e := os.Create(filename)
			if e != nil {
				err = e
				return
			}
			defer file.Close()

			_, err = file.Write(content)
			if err != nil {
				return
			}

		}
	}

	fmt.Printf("Saved: %s\n", filename)

	return
}

func splittedPage(root string, id int, pageurl string) (filename string, err error) {

	fmt.Printf("ファイアウォールのメッセージが出る場合、")
	fmt.Printf("キャンセル（不許可）を選んでも問題ありません\n")

	opt := agouti.Timeout(3)
	cmd := []string{tools.GetPath("phantomjs"), "--webdriver={{.Address}}", "--ignore-ssl-errors=true"}
	//cmd := []string{tools.GetPath("chromedriver"), "--port={{.Port}}"}

	driver := agouti.NewWebDriver("http://{{.Address}}", cmd, opt)

	if err = driver.Start(); err != nil {
		return
	}
	defer driver.Stop()

    page, err := driver.NewPage()
    if err != nil {
		return
    }
	err = page.SetImplicitWait(1000); if err != nil { return }
	err = page.SetPageLoad(60000); if err != nil { return }
	err = page.SetScriptTimeout(1000); if err != nil { return }

	if err = page.Navigate(pageurl); err != nil {
		return
	}

	content, err := ioutil.ReadFile("js/frame.js")
	if err != nil {
		return
	}
	script := string(content)

	var res interface{}
	err = page.RunScript(script, map[string]interface{}{}, &res)
	if err != nil {
		return
	}
	var lines []interface {}
	switch t := res.(type) {
		case []interface {}:
			lines = res.([]interface {})
		default:
			fmt.Printf("%v", res)
			err = fmt.Errorf("typeof res is not []interface {}, %v", t)
			return
	}

	var file_lines []string
	// each line
	for i, arr := range lines {
		var line []interface {}
		switch t := arr.(type) {
			case []interface {}:
				line = arr.([]interface {})
			default:
				fmt.Printf("%v", res)
				err = fmt.Errorf("typeof res[%d] is not []interface {}, %v", i, t)
				return
		}

		// join horizonally
		cmd_horiz := exec.Cmd{
			Path: tools.GetPath("convert"),
		}
		cmd_horiz.Args = append(cmd_horiz.Args, cmd_horiz.Path)
		cmd_horiz.Args = append(cmd_horiz.Args, "+append")

		for j, obj := range line {
			var url string
			switch t := obj.(type) {
				case string:
					url = obj.(string)
				default:
					fmt.Printf("%v", res)
					err = fmt.Errorf("typeof res[%d][%d] is not string, %v", i, j, t)
					return
			}
			cmd_horiz.Args = append(cmd_horiz.Args, url)
		}

		file_horiz := fmt.Sprintf("%s/%d-%d.jpg", root, id, i)
		cmd_horiz.Args = append(cmd_horiz.Args, file_horiz)

		_, err = cmd_horiz.Output()
		if err != nil {
			return
		}
		file_lines = append(file_lines, file_horiz)
	}

	// join vertically
	cmd_vert := exec.Cmd{
		Path: tools.GetPath("convert"),
	}
	cmd_vert.Args = append(cmd_vert.Args, cmd_vert.Path)
	cmd_vert.Args = append(cmd_vert.Args, "-append")
	for _, file := range file_lines {
		cmd_vert.Args = append(cmd_vert.Args, file)
	}
	filename = fmt.Sprintf("%s/%d.jpg", root, id)
	cmd_vert.Args = append(cmd_vert.Args, filename)
	_, err = cmd_vert.Output()
	if (err != nil) {
		return
	}

	for _, file := range file_lines {
		if err = os.Remove(file); err != nil {
			fmt.Printf("%s: %v\n", file, err)
		}
		cmd_vert.Args = append(cmd_vert.Args, file)
	}

	return
}

func findImageByNumber(root string, num int) (exist bool, filename string) {

	testpostfix := []string{jpg, ""}
	fileformat := []string{"%s/%d%s", "%s/%02d%s", "%s/%03d%s", "%s/%04d%s", "%s/%05d%s"}
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