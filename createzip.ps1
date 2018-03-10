
function run() {
	param($cmd)

	foreach($arg in $args) {
		if($arg.indexOf(" ") -ge 0) {
			$cmd += ' "' + $arg + '"'
		} else {
			$cmd += ' ' + $arg
		}
	}

	Write-Output $cmd
	$cmd += ' | Write-Host ; $?'
	$ret = iex $cmd

	[Console]::ResetColor()

	if(! $ret) {
		echo ""
		echo "Command failured: $cmd"
		exit
	}
}

# gui icon
run rc ./src_gui/res/icon.rc
# gui resource
run powershell -Command "cd ./src_gui/res/; fsc resx_gen.fs -o resx_gen.exe --standalone --nowarn:82"
run powershell -Command "cd ./src_gui/res/; ./resx_gen.exe ../../src/mangamuradl.go"
run powershell -Command "cd ./src_gui/res/; resgen mangamuradl-gui.resx mangamuradl-gui.resources"
# compile gui
run fsc ./src_gui/mangamuradl-gui.fs -o mangamuradl-gui.exe --resource:./src_gui/res/mangamuradl-gui.resources --win32res:./src_gui/res/icon.res --target:winexe --standalone --nowarn:82

# conpile cli
run go build -a -ldflags "-extldflags -static" src/mangamuradl.go

# create zip
$dir = "_tmp/mangamuradl"

if(Test-Path $dir) {
	run rm _tmp -Force -Recurse
}
run mkdir $dir
run mkdir $dir/js

run cp mangamuradl.exe     $dir
run cp mangamuradl-gui.exe $dir
run cp js/mmdl.js          $dir/js/mmdl.js
run cp js/frame.js         $dir/js/frame.js
run cp README.md           $dir/README.txt

$files = @(
	"_tmp/mangamuradl"
)

run Compress-Archive -F $files mangamuradl.zip

run rm _tmp -Force -Recurse

echo "ok"

