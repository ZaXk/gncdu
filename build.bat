@echo off
:: Linux builds
set GOOS=linux
set GOARCH=amd64
go build -o gncdu-linux-amd64
set GOARCH=arm64
go build -o gncdu-linux-arm64

:: macOS builds
set GOOS=darwin
set GOARCH=amd64
go build -o gncdu-darwin-amd64
set GOARCH=arm64
go build -o gncdu-darwin-arm64

:: Windows builds
set GOOS=windows
set GOARCH=amd64
go build -o gncdu-windows-amd64.exe
set GOARCH=arm64
go build -o gncdu-windows-arm64.exe
set GOARCH=386
go build -o gncdu-windows-386.exe

:: Reset environment variables
set GOOS=
set GOARCH= 