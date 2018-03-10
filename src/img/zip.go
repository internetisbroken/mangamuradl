// 180310 created

package img

import (
	"fmt"
	"os"
	"io"
	"regexp"
	"archive/zip"
	"errors"
	"database/sql"
)

func tryCreateZip(imgroot string, wrzip *zip.Writer, db *sql.DB) (count int, err error) {

	rows, err := db.Query("select pagenum from page order by pagenum")
	if err != nil {
		fmt.Printf("CreateZip: %v\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var pagenum int
		err = rows.Scan(&pagenum)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}

		ex, file := findImageByNumber(imgroot, pagenum)
		if ex {
			count++

			re := regexp.MustCompile(`[\\/]([^\\/]+)$`)
			m := re.FindStringSubmatch("./" + file)
			if len(m) < 2 {
				err = fmt.Errorf("Can't find basename: file")
				return
			}

			wr, e := wrzip.Create("img/" + m[1])
			if err != nil {
				err = e
				return
			}

			fp, e := os.Open(file)
			if e != nil {
				err = e
				return
			}
			_, e = io.Copy(wr, fp)
			fp.Close()
			if e != nil {
				err = e
				return
			}

		} else{
			msg := fmt.Sprintf("Not found: %s/%d.jpg", imgroot, pagenum)
			err = errors.New(msg)
			return
		}
	}
	err = rows.Err()
	if err != nil {
		return
	}
	return
}

func CreateZip(imgroot, zippath string, db *sql.DB) (err error) {

	zipfp, err := os.Create(zippath)
	if err != nil {
		return
	}
	fmt.Printf("Creating Zip[%s]\n", zippath)

	wr := zip.NewWriter(zipfp)
	cnt, err := tryCreateZip(imgroot, wr, db)
	wr.Close()
	zipfp.Close()

	if err != nil {
		return
	}
	if cnt <= 0 {
		if err = os.Remove(zippath); err != nil {
			return
		}
	}

	return
}
