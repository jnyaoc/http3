rem set GOOS=windows
set GOOS=linux
go build -trimpath -ldflags "-w" -o httpclient
rem set GOOS=linux
set GOOS=windows
go build -trimpath -ldflags "-w" -o httpclient.exe

