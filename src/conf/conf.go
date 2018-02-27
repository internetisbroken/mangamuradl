// 180228 created

package conf

import (
	"github.com/go-ini/ini"
)

var inifile = "mangamuradl.ini"
var cookie_key = "acookie4"

func GetCookie() (val string, err error) {
	return getIni(inifile, cookie_key)
}
func SetCookie(val string) (err error) {
	return setIni(inifile, cookie_key, val)
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
