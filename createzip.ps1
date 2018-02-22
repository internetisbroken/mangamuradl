
go build -a -ldflags "-extldflags -static" mangamuradl.go
fsc .\mangamuragui.fs --target:winexe --standalone
go build -a -ldflags "-extldflags -static" getcookie.go

cp README.md README.txt
$files = @(
	"mangamuradl.exe",
	"mangamuragui.exe",
	"getcookie.exe",
	"README.txt"
)
Compress-Archive -F $files mangamuradl.zip

rm README.txt