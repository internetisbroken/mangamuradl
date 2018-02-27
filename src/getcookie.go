// getcookie
//
// v1.0.0(180220) created
//

package main

import (
    "github.com/sclevine/agouti"
    "time"
	"fmt"
	"regexp"
	"strings"
	"github.com/go-ini/ini"
	"errors"
	"os"
)

func test() (cookie string, err error) {
	opt := agouti.Timeout(3)
	driver := agouti.ChromeDriver(opt)
	if err = driver.Start(); err != nil {
		return
	}
	defer driver.Stop()

    page, err := driver.NewPage()
    if err != nil {
		return
    }
	err = page.SetImplicitWait(1000); if err != nil { return }
	err = page.SetPageLoad(10000); if err != nil { return }
	err = page.SetScriptTimeout(1000); if err != nil { return }

	for j := 0; j < 10 ; j++ {
		if err = page.Navigate("http://mangamura.org/cap/g-recaptcha.html"); err != nil {
			return
		}
		var res interface{}
		err = page.RunScript(`
			function call2(code) {
				googleauth = code;
				$.post("/cap/chkauth.php", {googleauth: code}).done(function(message) {
					cookie = document.cookie;
				}).fail(function() {
				}).always(function() {
				});
			}
			function fixed() {
				var e = document.querySelector('.g-recaptcha[data-sitekey][data-callback]');
				if(e) {
					var funcname = e.getAttribute("data-callback");
					if(funcname) {
						var firstScript = document.getElementsByTagName('script')[0];
						var s1 = document.createElement('script');
						s1.type = 'text/javascript';
						s1.src = "https://code.jquery.com/jquery-2.2.4.min.js";
						firstScript.parentNode.insertBefore(s1, firstScript);
						eval(funcname + " = call2;");
						is_fixed = 1;
						return 1;
					}
				}
				return 0;
			}
			window.addEventListener("DOMContentLoaded", fixed, false);
			return fixed();
			`, map[string]interface{}{}, &res)

		if err != nil {
			return
		}

		resstr := fmt.Sprintf("%v", res)
		if resstr != "1" {
			goto NEXT_TRY
		}

		for i := 0; i < 120 ; i++ {
			time.Sleep(1 * time.Second)
			url, e := page.URL()
			if e != nil {
				err = e
				return
			}
			if url != "http://mangamura.org/cap/g-recaptcha.html" {
				goto NEXT_TRY
			}

			var res interface{}
			err = page.RunScript(`return cookie`, map[string]interface{}{}, &res)
			if err != nil {
				return
			}
			resstr := fmt.Sprintf("%v", res)

			re := regexp.MustCompile(`acookie4\s*=\s*([^;"'\s]*)`)
			m := re.FindStringSubmatch(resstr)
			if len(m) >=2 {
				cookie = m[1]
				return
			}

			err = page.RunScript(`return is_fixed`, map[string]interface{}{}, &res)
			if err != nil {
				return
			}
			resstr = fmt.Sprintf("%v", res)
			if resstr != "1" {
				goto NEXT_TRY
			}
		}
		err = errors.New("timeout")
		return

NEXT_TRY:
		err = page.RunScript(`document.write("5秒後リロードします")`, map[string]interface{}{}, nil)
		if err != nil {
			return
		}
		for k := 0; k < 5; k++ {
			time.Sleep(1 * time.Second)
			_, err = page.URL()
			if err != nil {
				return
			}
		}
	}

	err = errors.New("Retry exceeded")
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

/*
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
*/

func main() {

	cookie, err := test()
	if err != nil {
		fmt.Printf("%v\n", err)
		if strings.Index(err.Error(), "failed to start service") >= 0 {
			fmt.Printf("\nchromedriverとchromeが必要です\n")
			fmt.Printf("chromedriverは以下からダウンロードしてから\n")
			fmt.Printf("chromedriver.exeをgetcookie.exeと同じディレクトリに置いてください\n")
			fmt.Printf("\n%s\n", "https://sites.google.com/a/chromium.org/chromedriver/downloads")
			fmt.Printf("リンク先の「Latest Release: ChromeDriver X.XX」-->「chromedriver_win32.zip」\n")
		}
		os.Exit(1)
	}
	if cookie != "" {
		err = UpdateIni("mangamuradl.ini", "acookie4", cookie)
		if err == nil {
			os.Exit(0)
		}

		fmt.Printf("%v\n", err)
	}
	os.Exit(1)
}
