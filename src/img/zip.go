// 180310 created
// 180315 changed go archive/zip to 7za

package img

import (
	"fmt"
	"os/exec"
	"errors"
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
			// [<file_names>...]
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

	fmt.Printf("Creating ZIP[%s]\n", zippath)
	_, err = command.Output()

	return
}
