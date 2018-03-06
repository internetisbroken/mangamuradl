// 180228 created
// 180306 add os.Remove splitted pages

package img

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"errors"
	"encoding/base64"
	"github.com/sclevine/agouti"
	"../httpwrap"
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
	driver := agouti.PhantomJS(opt)
	//driver := agouti.ChromeDriver(opt)
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

	var res interface{}
	err = page.RunScript(`
		var a = document.querySelectorAll("img");
		var data = {};
		for(var i = 0; i < a.length; i++) {
			var element = a[i];
			var t = 0, left = 0;
			do {
				t += element.offsetTop  || 0;
				left += element.offsetLeft || 0;
				element = element.offsetParent;
			} while(element);
			if(! data[t]) {
				data[t] = {};
			}
			data[t][left] = a[i].src;
		}

		var comp = function(a, b) {
			return((a*1) - (b*1));
		}

		var k0 = [];
		for(var k in data) {
			k0.push(k);
		}
		k0 = k0.sort(comp);

		var line = [];
		for(var i = 0; i < k0.length; i++) {
			var key_0 = k0[i];
			var k1 = [];
			for(var t in data[key_0]) {
				k1.push(t);
			}
			k1 = k1.sort(comp);
			var urls = [];
			for(var j = 0; j < k1.length; j++) {
				var key_1 = k1[j];
				urls.push(data[key_0][key_1]);
			}
			line.push(urls);
		}
		return JSON.stringify(line);
		`, map[string]interface{}{}, &res)
	if err != nil {
		return
	}

	resstr := fmt.Sprintf("%v", res)

	re := regexp.MustCompile(`\[([^\[]*?)\]`)
	m := re.FindAllStringSubmatch(resstr, -1)

	var filenames []string
	for i := 0; i < len(m); i++ {
		re := regexp.MustCompile(`"(.+?)"`)
		m0 := re.FindAllStringSubmatch(m[i][1], -1)

		command := exec.Cmd{
			Path: "./convert",
		}
		command.Args = append(command.Args, command.Path)
		command.Args = append(command.Args, "+append")
		for j:= 0; j < len(m0); j++ {
			command.Args = append(command.Args, m0[j][1])
		}
		filename_0 := fmt.Sprintf("%s/%d-%d.jpg", root, id, i)
		command.Args = append(command.Args, filename_0)

		_, e := command.Output()
		if e != nil {
			err = e
			return
		}
		filenames = append(filenames, filename_0)
	}

	command := exec.Cmd{
		Path: "./convert",
	}
	command.Args = append(command.Args, command.Path)
	command.Args = append(command.Args, "-append")
	for i := 0; i < len(filenames); i++ {
		command.Args = append(command.Args, filenames[i])
	}
	filename = fmt.Sprintf("%s/%d.jpg", root, id)
	command.Args = append(command.Args, filename)
	_, err = command.Output()
	if (err != nil) {
		return
	}

	for i := 0; i < len(filenames); i++ {
		if err = os.Remove(filenames[i]); err != nil {
			fmt.Printf("%v\n", filenames[i])
		}
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