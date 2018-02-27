// 180228 created

package httpwrap

import (
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

func httpReq(req *http.Request) (content []byte, err error) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/63.0.3239.132 Safari/537.36")
	req.Header.Set("Origin", "http://mangamura.org");

	//---req.Close = true
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	content, err = ioutil.ReadAll(res.Body)
	io.Copy(ioutil.Discard, res.Body)
	res.Body.Close()
	return
}

// post text then return as []byte
func HttpPostByte(url string, data string) (content []byte, err error) {
	reader := strings.NewReader(data)
	req, err := http.NewRequest("POST", url, reader);
	if err != nil {
		return
	}

	return httpReq(req)
}

// post text then return as string
func HttpPostText(url string, data string) (content string, err error) {
	bytes, err := HttpPostByte(url, data)
	if err != nil {
		return
	}
	content = string(bytes)
	return
}

// get then return as []byte
func HttpGetByte(url string) (content []byte, err error) {
	req, err := http.NewRequest("GET", url, nil);
	if err != nil {
		return
	}

	return httpReq(req)
}

// get then return as string
func HttpGetText(url string) (content string, err error) {
	bytes, err := HttpGetByte(url)
	if err != nil {
		return
	}
	content = string(bytes)
	return
}
