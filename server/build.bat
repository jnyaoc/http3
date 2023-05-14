rem set GOOS=windows
set GOOS=linux
go build -trimpath -ldflags "-w" -o httpserver
rem set GOOS=linux
set GOOS=windows
go build -trimpath -ldflags "-w" -o httpserver.exe

