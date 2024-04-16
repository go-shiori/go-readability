# build x86 so
GOOS=linux GOARCH=amd64 go build -buildmode=c-shared -o libparser-x86.so scripts/export.go