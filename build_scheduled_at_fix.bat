@echo off
cd /d "c:\Users\aqilz\Documents\go-whatsapp-web-multidevice-main\src"
set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o ../whatsapp main.go
cd ..
git add .
git commit -m "Fix: Standardize date filtering to use scheduled_at column across all APIs - Changed Report NEW and Progress NEW from created_at to scheduled_at - All three pages (Detail Sequences, Report NEW, Progress NEW) now use same column - This fixes data mismatch where Dashboard showed 886 but Report showed 345"
git push
