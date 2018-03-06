
function ng() {
	echo "Command failured"
	exit
}
$dir = "_tmp/mangamuradl"

if(Test-Path $dir) {
	rm _tmp -Force -Recurse ; if(!$?){ng}
}
mkdir $dir ; if(!$?){ng}
mkdir $dir/js ; if(!$?){ng}

go build -a -ldflags "-extldflags -static" src/mangamuradl.go ; if(!$?){ng}
fsc .\mangamuragui.fs --target:winexe --standalone --nowarn:82 ; if(!$?){ng}

cp mangamuradl.exe $dir ; if(!$?){ng}
cp js/mmdl.js $dir/js/mmdl.js ; if(!$?){ng}
cp mangamuragui.exe $dir ; if(!$?){ng}
cp README.md $dir/README.txt ; if(!$?){ng}

$files = @(
	"_tmp/mangamuradl"
)

Compress-Archive -F $files mangamuradl.zip ; if(!$?){ng}

rm _tmp -Force -Recurse ; if(!$?){ng}

echo "ok"

