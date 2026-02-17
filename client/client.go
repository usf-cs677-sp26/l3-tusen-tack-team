package main

import (
	"crypto/md5"
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
		log.Fatalln(err)
	}

	file, err := os.Open(fileName)
	if err != nil {
		log.Println("Error opening file: ", err)
		return 1
	}
	md5Hash := md5.New()
	buf := make([]byte, 1024*1024) // 1MB buffer for better performance
	io.CopyBuffer(md5Hash, file, buf)
	checksum := md5Hash.Sum(nil)
	file.Close()

	// Tell the server we want to store this file
	msgHandler.SendStorageRequest(fileName, uint64(info.Size()), checksum)
	if ok, _ := msgHandler.ReceiveResponse(); !ok {
		return 1
	}

	file, err = os.Open(fileName)
	if err != nil {
		log.Println("Error opening file: ", err)
		return 1
	}
	buf := make([]byte, 1024*1024)
	_, err = io.CopyBuffer(msgHandler, io.LimitReader(file, info.Size()), buf)
	file.Close()
	if err != nil {
		log.Println("Error sending file data: ", err)
		return 1
	}

	if ok, _ := msgHandler.ReceiveResponse(); !ok {
		return 1
	}

	fmt.Println("Storage complete!")
	return 0
}

func get(msgHandler *messages.MessageHandler, fileName string, dir string) int {
	fmt.Println("GET", fileName)

	localPath := filepath.Join(dir, fileName)
	file, err := os.OpenFile(localPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		log.Println(err)
		return 1
	}

	msgHandler.SendRetrievalRequest(fileName)
	ok, _, size := msgHandler.ReceiveRetrievalResponse()
	if !ok {
		return 1
	}

	md5 := md5.New()
	w := io.MultiWriter(file, md5)
	buf := make([]byte, 1024*1024) // 1MB buffer for better performance
	_, err = io.CopyBuffer(w, io.LimitReader(msgHandler, int64(size)), buf)
	if err != nil {
		log.Println("Error receiving file data: ", err)
		file.Close()
		return 1
	}
	file.Close()

	clientCheck := md5.Sum(nil)
	checkMsg, err := msgHandler.Receive()
	if err != nil {
		log.Println("Error receiving checksum: ", err)
		return 1
	}
	serverCheck := checkMsg.GetChecksum().Checksum

	if util.VerifyChecksum(serverCheck, clientCheck) {
		log.Println("Successfully retrieved file.")
		return 0
	} else {
		log.Println("FAILED to retrieve file. Invalid checksum.")
		return 1
	}
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
