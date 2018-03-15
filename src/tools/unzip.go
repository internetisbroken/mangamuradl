// 180306 created
// 180315 add trim prefix

package tools

import (
	"os"
	"io"
	"path/filepath"
	"archive/zip"
	"regexp"
	"strings"
)

func getPrefix(src string) (prefix string, err error) {
	r, err := zip.OpenReader(src)
	if err != nil {
		return
	}
	defer r.Close()

	var flist []string
	for _, f := range r.File {
		if (! f.FileInfo().IsDir()) {
			flist = append(flist, f.Name)
		}
	}

	re := regexp.MustCompile(`^(.+?/)`)

	for {
		newlist := []string{}
		for _, f := range flist {
			if strings.Compare(f, "") != 0 {
				newlist = append(newlist, f)
			}
		}
		if len(newlist) <= 0 {
			return
		}
		flist = newlist

		ma := re.FindStringSubmatch(flist[0])
		if len(ma) >= 2 {
			test := ma[1]
			for i := 0; i < len(flist) ; i++ {
				//fmt.Printf("flist[%d] is %s\n", i, flist[i])

				if strings.Index(flist[i], test) != 0 {
					return
				}
				flist[i] = strings.TrimPrefix(flist[i], test)
			}
			prefix += test
			//fmt.Printf("%s is prefix\n", prefix)
		} else {
			return
		}
	}
	return
}


// stackoverflow.com/questions/20357223
func unzip(src, dest string, trimPrefix bool) (err error) {
	var prefix string
	if trimPrefix {
		prefix, err = getPrefix(src)
		if err != nil {
			return
		}
	}

    r, err := zip.OpenReader(src)
    if err != nil {
        return
    }
    defer r.Close()

    os.MkdirAll(dest, 0755)

    // Closure to address file descriptors issue with all the deferred .Close() methods
    extractAndWriteFile := func(f *zip.File) error {
        rc, err := f.Open()
        if err != nil {
            return err
        }
        defer func() {
            if err := rc.Close(); err != nil {
                panic(err)
            }
        }()

		fname := strings.TrimPrefix(f.Name, prefix)
        path := filepath.Join(dest, fname)

        if f.FileInfo().IsDir() {
            os.MkdirAll(path, f.Mode())
        } else {
            os.MkdirAll(filepath.Dir(path), f.Mode())
            f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
            if err != nil {
                return err
            }
            defer func() {
                if err := f.Close(); err != nil {
                    panic(err)
                }
            }()

            _, err = io.Copy(f, rc)
            if err != nil {
                return err
            }
        }
        return nil
    }

    for _, f := range r.File {
        err := extractAndWriteFile(f)
        if err != nil {
            return err
        }
    }

    return nil
}
