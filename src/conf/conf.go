// 180228 created
// 180315 change default pdf(off)/zip(on)

package conf

import (
	"github.com/go-ini/ini"
)

var inifile = "mangamuradl.ini"
var cookie_key = "acookie4"
var pdf_key = "pdf"
var zip_key = "zip"

// Cookie
func GetCookie() (val string, err error) {
	return getIni(inifile, cookie_key)
}
func SetCookie(val string) (err error) {
	return setIni(inifile, cookie_key, val)
}

// PDf
func IsPdf() (bool) {
	val, _ := getIni(inifile, pdf_key)
	if val == "1" {
		return true
	}
	// default: false
	if val == "" {
		return false
	}
	return false
}
func SetPdf(val bool) (err error) {
	if val {
		return setIni(inifile, pdf_key, "1")
	}
	return setIni(inifile, pdf_key, "0")
}

// Zip
func IsZip() (bool) {
	val, _ := getIni(inifile, zip_key)
	if val == "1" {
		return true
	}
	// default: true
	if val == "" {
		return true
	}
	return false
}
func SetZip(val bool) (err error) {
	if val {
		return setIni(inifile, zip_key, "1")
	}
	return setIni(inifile, zip_key, "0")
}

func getIni(filename, key string) (val string, err error) {
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

func setIni(filename, key, val string) (err error) {
	cfg, err := ini.LooseLoad(filename)
	if err != nil {
		return
	}
	cfg.Section("").NewKey(key, val)
	err = cfg.SaveTo(filename)
	return
}
