package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"github.com/rclone/rclone/backend/onedrive/quickxorhash"
)

// go mod init data-migration
// go get github.com/rclone/rclone
// go build -o quickxorhash quickxorhash.go
// go build -o quickxorhash.exe quickxorhash.go
// for windows
// $env:GOOS="windows"; $env:GOARCH="amd64"; go build -o quickxorhash.exe quickxorhash.go

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: quickxorhash <file>")
		os.Exit(1)
	}

	filePath := os.Args[1]
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		os.Exit(1)
	}
	defer file.Close()

	hasher := quickxorhash.New()
	if _, err := io.Copy(hasher, file); err != nil {
		fmt.Println("Error reading file:", err)
		os.Exit(1)
	}

	hash := hasher.Sum(nil)
	// Encode the hash in base64
	base64Hash := base64.StdEncoding.EncodeToString(hash)
	fmt.Println(base64Hash)
}
