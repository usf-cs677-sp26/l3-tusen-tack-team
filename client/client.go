package main

import (
	"crypto/md5"
	"errors"
	"file-transfer/messages"
	"file-transfer/util"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

func put(msgHandler *messages.MessageHandler, fileName string) int {
	fmt.Println("PUT", fileName)

	// Get file size and make sure it exists
	info, err := os.Stat(fileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("Source file does not exist: %s", fileName)
		} else {
			log.Printf("Could not stat source file %s: %v", fileName, err)
		}
		return 1
	}
	if !info.Mode().IsRegular() {
		log.Printf("Source path is not a regular file: %s", fileName)
		return 1
	}

	// Open once, hash once, then seek back to start for upload.
	file, err := os.Open(fileName)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			log.Printf("Permission denied opening %s", fileName)
		} else {
			log.Printf("Could not open source file %s: %v", fileName, err)
		}
		return 1
	}
	defer file.Close()

	hasher := md5.New()
	n, err := io.CopyN(hasher, file, info.Size())
	if err != nil {
		log.Printf("Failed to compute checksum: %v", err)
		return 1
	}
	if n != info.Size() {
		log.Printf("Failed to compute checksum: hashed %d of %d bytes", n, info.Size())
		return 1
	}
	checksum := hasher.Sum(nil)
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		log.Printf("Failed to rewind source file: %v", err)
		return 1
	}

	// Tell the server we want to store this file
	if err := msgHandler.SendStorageRequest(fileName, uint64(info.Size()), checksum); err != nil {
		log.Printf("Failed to send storage request: %v", err)
		return 1
	}

	ok, msg := msgHandler.ReceiveResponse()
	if !ok {
		log.Printf("Put request rejected by server: %s", msg)
		return 1
	}

	n, err = io.CopyN(msgHandler, file, info.Size())
	if err != nil {
		log.Printf("Failed to upload file: %v", err)
		return 1
	}
	if n != info.Size() {
		log.Printf("Short upload: wrote %d of %d bytes", n, info.Size())
		return 1
	}
	if ok, msg := msgHandler.ReceiveResponse(); !ok {
		log.Printf("Upload rejected by server: %s", msg)
		return 1
	}

	fmt.Println("Storage complete!")
	return 0
}

func get(msgHandler *messages.MessageHandler, fileName, destPath string) int {
	fmt.Println("GET", fileName)

	if err := msgHandler.SendRetrievalRequest(fileName); err != nil {
		log.Printf("Failed to send retrieval request: %v", err)
		return 1
	}

	ok, msg, size := msgHandler.ReceiveRetrievalResponse()
	if !ok {
		log.Printf("Get request rejected by server: %s", msg)
		return 1
	}

	destFile := filepath.Join(destPath, filepath.Base(fileName))
	file, err := os.OpenFile(destFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		switch {
		case errors.Is(err, os.ErrExist):
			log.Printf("destination already exists: %s", destFile)
		case errors.Is(err, os.ErrPermission):
			log.Printf("no permission to write: %s", destFile)
		case errors.Is(err, os.ErrNotExist):
			log.Printf("destination directory does not exist: %s", destPath)
		default:
			log.Printf("cannot create destination file %s: %v", destFile, err)
		}
		return 1
	}
	defer file.Close()

	md5 := md5.New()
	w := io.MultiWriter(file, md5)
	n, err := io.CopyN(w, msgHandler, int64(size))
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			log.Printf("download interrupted: got %d/%d bytes", n, size)
		} else {
			log.Printf("download failed after %d/%d bytes: %v", n, size, err)
		}
		_ = os.Remove(destFile)
		return 1
	}
	if n != int64(size) {
		log.Printf("short download: got %d/%d bytes", n, size)
		_ = os.Remove(destFile)
		return 1
	}
	file.Close()

	clientCheck := md5.Sum(nil)
	checkMsg, err := msgHandler.Receive()
	if err != nil {
		log.Printf("Failed to receive server checksum: %v", err)
		_ = os.Remove(destFile)
		return 1
	}

	checksumMsg := checkMsg.GetChecksum()
	if checksumMsg == nil {
		log.Printf("Protocol error: expected checksum message from server")
		_ = os.Remove(destFile)
		return 1
	}

	serverCheck := checksumMsg.Checksum
	if util.VerifyChecksum(serverCheck, clientCheck) {
		log.Println("Successfully retrieved file.")
	} else {
		log.Println("FAILED to retrieve file. Invalid checksum.")
		_ = os.Remove(destFile)
		return 1
	}
	return 0
}

func main() {
	if len(os.Args) < 4 {
		fmt.Printf("Not enough arguments. Usage: %s server:port put|get file-name [download-dir]\n", os.Args[0])
		os.Exit(1)
	}

	host := os.Args[1]
	conn, err := net.Dial("tcp", host)
	if err != nil {
		log.Fatalln(err.Error())
		return
	}
	msgHandler := messages.NewMessageHandler(conn)
	defer conn.Close()

	action := strings.ToLower(os.Args[2])
	if action != "put" && action != "get" {
		log.Fatalln("Invalid action", action)
	}

	fileName := os.Args[3]

	dir := "."
	if len(os.Args) >= 5 {
		dir = os.Args[4]
	}
	openDir, err := os.Open(dir)
	if err != nil {
		log.Fatalln(err)
	}
	openDir.Close()

	if action == "put" {
		os.Exit(put(msgHandler, fileName))
	} else if action == "get" {
		os.Exit(get(msgHandler, fileName, dir))
	}
}
