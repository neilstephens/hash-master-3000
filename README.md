
# Build
go mod init HashMaster3000
go mod tidy
go build -ldflags "-s -w"

# Package
go install fyne.io/tools/cmd/fyne@latest

## Package for Android 
fyne package --id link.multifarious.hm3k --release --icon HM3k.png -os android

## Package for Linux
fyne package --id link.multifarious.hm3k --release --icon HM3k.png -os linux
