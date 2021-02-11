go fmt
go build -ldflags "-s -w"
tools\upx.exe pixelSequencer.exe