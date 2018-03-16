// 180310 created
// 180315 changed go archive/zip to 7za
// 180316 support webp

package img

import (
	"fmt"
	"os"
	"os/exec"
	"errors"
	"strings"
	"database/sql"
	"../tools"
)

func CreateZip(imgroot, zippath string, db *sql.DB) (err error) {

	exe, err := tools.Get7za()
	if err != nil {
		return
	}

	// index: pagenum
	rows, err := db.Query("select pagenum from page order by pagenum")
	if err != nil {
		fmt.Printf("CreateZip: %v\n", err)
		return
	}
	defer rows.Close()

	// remove zip if exists
	_, err = os.Stat(zippath)
	if err == nil {
		err = os.Remove(zippath)
		if err != nil {
			return
		}
	}

	//Usage: 7za <command> [<switches>...] <archive_name> [<file_names>...] [<@listfiles...>]
	command := exec.Cmd{
		Path: exe,
	}
	command.Args = append(command.Args, command.Path)

	// <command> a: Add files to archive
	command.Args = append(command.Args, "a")

	// <archive_name>
	command.Args = append(command.Args, zippath)

	var count int
	for rows.Next() {
		var pagenum int
		err = rows.Scan(&pagenum)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}

		ex, file := FindImageByNumber(imgroot, pagenum)
		if ex {
			count++
			if strings.HasSuffix(file, ".webp") {
				// convert webp to jpg
				jpgfile := strings.TrimSuffix(file, ".webp") + ".jpg"
				cvt, e := tools.GetConvert()
				if e != nil {
					err = e
					return
				}
				fmt.Printf("Converting: %s -> %s\n", file, jpgfile)
				cmd := exec.Cmd{Path: cvt}
				cmd.Args = append(cmd.Args, cmd.Path)
				cmd.Args = append(cmd.Args, file)
				cmd.Args = append(cmd.Args, jpgfile)
				_, err = cmd.Output()
				if err != nil {
					return
				}

				// [<file_names>...]
				command.Args = append(command.Args, jpgfile)
			} else {
				// jpg
				// [<file_names>...]
				command.Args = append(command.Args, file)
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

	fmt.Printf("Creating ZIP[%s]\n", zippath)
	_, err = command.Output()

	return
}
