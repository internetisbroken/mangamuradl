function ng() {
	echo "Command failured"
	exit
}
go build -a -ldflags "-extldflags -static" src/mangamuradl.go ; if(!$?){ng}
fsc .\mangamuragui.fs --target:winexe --standalone ; if(!$?){ng}
go build -a -ldflags "-extldflags -static" src/getcookie.go ; if(!$?){ng}

cp README.md README.txt ; if(!$?){ng}
$files = @(
	"mangamuradl.exe",
	"mangamuragui.exe",
	"getcookie.exe",
	"README.txt"
)
Compress-Archive -F $files mangamuradl.zip ; if(!$?){ng}

rm README.txt ; if(!$?){ng}

echo "ok"

