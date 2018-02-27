// 180228 created

package img

import (
	"fmt"
	"os/exec"
	"errors"
	"database/sql"
	"../tools"
)

func CreatePdf(imgroot, pdfpath string, db *sql.DB) (err error) {

	err = tools.DownloadImageMagic()
	if err != nil {
		return
	}

	// index: pagenum
	rows, err := db.Query("select pagenum from page order by pagenum")
	if err != nil {
		fmt.Printf("CreatePdf: %v\n", err)
		return
	}
	defer rows.Close()


	command := exec.Cmd{
		Path: "./convert",
	}
	command.Args = append(command.Args, command.Path)

	var count int
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
			command.Args = append(command.Args, file)
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

	command.Args = append(command.Args, pdfpath)

	fmt.Printf("Creating PDF[%s]\n", pdfpath)
	_, err = command.Output()

	return
}
