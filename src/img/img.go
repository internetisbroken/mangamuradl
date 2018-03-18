// 180228 created
// 180306 add os.Remove splitted pages
// 180310 remove javascript code
// 180315 update tools
// 180315 add DownloadFrameImage
// 180316 add webp support
// 180316 file name zero padding
// 180318 support image formats

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
var webp = ".webp"
var png = ".png"
var bmp = ".bmp"
var gif = ".gif"
var tiff = ".tiff"
var pdf = ".pdf"
var testpostfix = []string{jpg, webp, png, bmp, gif, tiff, pdf}


func DownloadImage(root string, pagenum int, url string, is_frame, is_blob bool, base64str string, numLength int) (err error) {

	exists, testname := FindImageByNumber(root, pagenum)
	if exists {
		if testname == "" {
			//fmt.Printf("Skip: %s\n", testname)
		}
		return
	}

	var filename string
	if is_frame {
		fmt.Printf("Skip: page[%d](frame)\n", pagenum)
		return

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

			} else if (
				// WebP
				'R' == content[0] &&
				'I' == content[1] &&
				'F' == content[2] &&
				'F' == content[3] &&
				'W' == content[8] &&
				'E' == content[9] &&
				'B' == content[10] &&
				'P' == content[11] ) {
				postfix = webp

			} else if
				0x89 == content[0] &&
				0x50 == content[1] &&
				0x4e == content[2] &&
				0x47 == content[3] &&
				0x0d == content[4] &&
				0x0a == content[5] &&
				0x1a == content[6] &&
				0x0a == content[7] {
				postfix = png

			} else if
				'B' == content[0] &&
				'M' == content[1] {
				postfix = bmp

			} else if
				'G' == content[0] &&
				'I' == content[1] &&
				'F' == content[2] &&
				'8' == content[3] &&
				('7' == content[4] || '9' == content[4]) &&
				'a' == content[5] {
				postfix = gif

			} else if (
				0x49 == content[0] &&
				0x49 == content[1] &&
				0x2A == content[2] &&
				0x00 == content[3] ) || (
				0x4D == content[0] &&
				0x4D == content[1] &&
				0x00 == content[2] &&
				0x2A == content[3]) {
				postfix = tiff

			} else if
				'%' == content[0] &&
				'P' == content[1] &&
				'D' == content[2] &&
				'F' == content[3] {
				postfix = pdf

			} else {
				errstr := fmt.Sprintf("Unknown format: %d(%s), header is %x %x %x %x\n",
					pagenum, url, content[0], content[1], content[2], content[3])
				err = errors.New(errstr)
				return
			}

			fmtFile := fmt.Sprintf(`%%s/%%0%dd%%s`, numLength)
			filename = fmt.Sprintf(fmtFile, root, pagenum, postfix)
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

func DownloadFrameImage(root string, pagenum int, url string, page *agouti.Page, numLength int) (filename string, err error) {
	err = page.SetImplicitWait(1000); if err != nil { return }
	err = page.SetPageLoad(60000); if err != nil { return }
	err = page.SetScriptTimeout(1000); if err != nil { return }

	if err = page.Navigate(url); err != nil {
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

	exeCv, err := tools.GetConvert()
	if err != nil {
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
			Path: exeCv,
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

		fmt_horiz := fmt.Sprintf(`%%s/%%0%dd-%%d.jpg`, numLength)
		file_horiz := fmt.Sprintf(fmt_horiz, root, pagenum, i)
		cmd_horiz.Args = append(cmd_horiz.Args, file_horiz)

		_, err = cmd_horiz.Output()
		if err != nil {
			return
		}
		file_lines = append(file_lines, file_horiz)
	}

	// join vertically
	cmd_vert := exec.Cmd{
		Path: exeCv,
	}
	cmd_vert.Args = append(cmd_vert.Args, cmd_vert.Path)
	cmd_vert.Args = append(cmd_vert.Args, "-append")
	for _, file := range file_lines {
		cmd_vert.Args = append(cmd_vert.Args, file)
	}
	fmtFile := fmt.Sprintf(`%%s/%%0%dd.jpg`, numLength)
	filename = fmt.Sprintf(fmtFile, root, pagenum)
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

	fmt.Printf("Saved: %s\n", filename)

	return
}

func FindImageByNumber(root string, num int) (exist bool, filename string) {

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