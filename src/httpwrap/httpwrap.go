// 180228 created

package httpwrap

import (
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"compress/gzip"
)

func httpReq(req *http.Request) (content []byte, err error) {

	//---req.Close = true
	ce := req.Header.Get("Content-Encoding")
	if ce == "" {
		req.Header.Set("Accept-Encoding", "gzip")
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	enc := res.Header.Get("Content-Encoding")
	if (enc == "gzip") && (! res.Uncompressed) {
		r, e := gzip.NewReader(res.Body)
		if e != nil {
			err = e
			return
		}
		defer r.Close()
		content, err = ioutil.ReadAll(r)
		io.Copy(ioutil.Discard, r)
	} else {
		content, err = ioutil.ReadAll(res.Body)
	}
	io.Copy(ioutil.Discard, res.Body)

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

/////////////////////////////////////////////////
// with headers
/////////////////////////////////////////////////
func httpReqH(req *http.Request, headers map[string] string) (content []byte, err error) {
	for k, v := range headers {
		//fmt.Printf("key[%s] value[%s]\n", k, v)
		req.Header.Set(k, v)
	}
	return httpReq(req)
}

// post text then return as []byte
func HttpPostByteH(url string, data string, headers map[string] string) (content []byte, err error) {
	reader := strings.NewReader(data)
	req, err := http.NewRequest("POST", url, reader);
	if err != nil {
		return
	}

	return httpReqH(req, headers)
}

// post text then return as string
func HttpPostTextH(url string, data string, headers map[string] string) (content string, err error) {
	bytes, err := HttpPostByteH(url, data, headers)
	if err != nil {
		return
	}
	content = string(bytes)
	return
}

// get then return as []byte
func HttpGetByteH(url string, headers map[string] string) (content []byte, err error) {
	req, err := http.NewRequest("GET", url, nil);
	if err != nil {
		return
	}

	return httpReqH(req, headers)
}

// get then return as string
func HttpGetTextH(url string, headers map[string] string) (content string, err error) {
	bytes, err := HttpGetByteH(url, headers)
	if err != nil {
		return
	}
	content = string(bytes)
	return
}


