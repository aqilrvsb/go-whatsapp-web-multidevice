@echo off
cd C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src
set CGO_ENABLED=0
go build -v -o ..\whatsapp.exe main.go