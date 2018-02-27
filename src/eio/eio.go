// 180228 created

package eio

import (
	"strings"
	"time"
	"strconv"
	"errors"
)

// engin.io
// https://github.com/socketio/engine.io-protocol
// https://github.com/socketio/socket.io-protocol
func Parse(bytes []byte, pos *int, eio_type, sio_type *int, payload *[]byte) (finish bool, err error) {
	//bytes := []byte(content)
	//pos := 0

	if *pos >= len(bytes) {
		finish = true
		err = errors.New("No data parsed")
		return
	}
	i := *pos
	// pos
	// 10:42xxxxxxxxxx
	// i
	for {
		if (len(bytes) > i) && '0' <= bytes[i] && bytes[i] <= '9' {
			i++
		} else {
			break
		}
	}
	// Now,
	// pos
	// 10:42xxxxxxxxxx
	//   i

	if len(bytes) <= i || bytes[i] != ':' {
		err = errors.New("error parsing engine.io")
		return
	}
	size, e := strconv.Atoi(string(bytes[*pos:i]))
	if e != nil {
		err = e
		return
	}
	//fmt.Printf("[%d %d] %v\n", pos, i, size)

	*pos = i + 1
	// Now,
	//    pos
	// 10:42xxxxxxxxxx
	//   i

	i = i + 1 + size
	// Now,
	//    pos
	// 10:42xxxxxxxx12
	//              i
	if i > len(bytes) {
		err = errors.New("error parsing engine.io")
		return
	}
	if size >= 1 {
		if '0' <= bytes[*pos] && bytes[*pos] <= '9' {
			*eio_type = int(bytes[*pos] - '0')
			*pos++
		} else {
			err = errors.New("engine.io packet type missing")
			return
		}
	} else {
		err = errors.New("size 0 engine.io packet(eio_type)")
		return
	}

	if *eio_type == 4 {
		if size >= 2 {
			if '0' <= bytes[*pos] && bytes[*pos] <= '9' {
				*sio_type = int(bytes[*pos] - '0')
				*pos++
			} else {
				err = errors.New("socket.io type missing")
				return
			}
		} else {
			err = errors.New("size 0 socket.io packet(sio_type)")
			return
		}
	}

	*payload = bytes[*pos:i]

	*pos = i

	if *pos >= len(bytes) {
		finish = true
	}

	return
}

// yeast
func EncodedTime() (encoded string) {
	// config
	str := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-_"
	var length int64
	length = 64

	var num int64
	num = time.Now().UnixNano()/(1000*1000)

	alphas := strings.Split(str, "")

	for {
		encoded = alphas[num % length] + encoded

		num = num / length
		if num <= 0 {
			break
		}
	}

	return
}