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
)

func handleStorage(msgHandler *messages.MessageHandler, request *messages.StorageRequest) {
	log.Println("Attempting to store", request.FileName)

	// Check storage space
	fileSize := request.Size
	freeSpace, err := util.GetStorageSize(".")
	if freeSpace < request.Size {
		msgHandler.SendResponse(false, "Not enough disk space")
		return
	}

	file, err := os.OpenFile(request.FileName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		msgHandler.SendResponse(false, err.Error())
		msgHandler.Close()
		return
	}

	msgHandler.SendResponse(true, "Ready for data")
	md5 := md5.New()
	w := io.MultiWriter(file, md5)

	/* Write and checksum as we go */
	n, err := io.CopyN(w, msgHandler, int64(fileSize))
	if err != nil {
		msgHandler.SendResponse(false, err.Error())
		os.Remove(request.FileName)
		return
	}
	if n != int64(fileSize) {
		msgHandler.SendResponse(false, "short copy")
		os.Remove(request.FileName)
		return
	}
	file.Close()

	serverCheck := md5.Sum(nil)

	clientChecksum := request.GetChecksum()

	if util.VerifyChecksum(serverCheck, clientChecksum) {
		msgHandler.SendResponse(true, "Checksum match! Your file has been stored")
	} else {
		msgHandler.SendResponse(false, "Uh-oh! Checksum did not match! Your file was not stored")
		os.Remove(request.FileName)
	}
}

func handleRetrieval(msgHandler *messages.MessageHandler, request *messages.RetrievalRequest) {
	log.Println("Attempting to retrieve", request.FileName)

	// Get file size and make sure it exists
	info, err := os.Stat(request.FileName)
	if err != nil {
		msgHandler.SendResponse(false, "Failed to stat file")
		return
	}

	msgHandler.SendRetrievalResponse(true, "Ready to send", uint64(info.Size()))

	file, _ := os.Open(request.FileName)
	md5 := md5.New()
	w := io.MultiWriter(msgHandler, md5)
	n, err := io.CopyN(w, file, info.Size())
	if err != nil {
		msgHandler.SendResponse(false, "Failed to retrieve file")
		return
	}
	if n != info.Size() {
		msgHandler.SendResponse(false, "short copy")
		return
	}
	file.Close()

	checksum := md5.Sum(nil)
	msgHandler.SendChecksumVerification(checksum)
}

func handleClient(msgHandler *messages.MessageHandler) {
	defer msgHandler.Close()

	for {
		wrapper, err := msgHandler.Receive()
		if err != nil {
			log.Println(err)
			return
		}

		switch msg := wrapper.Msg.(type) {
		case *messages.Wrapper_StorageReq:
			handleStorage(msgHandler, msg.StorageReq)
			continue
		case *messages.Wrapper_RetrievalReq:
			handleRetrieval(msgHandler, msg.RetrievalReq)
			continue
		case nil:
			log.Println("Received an empty message, terminating client")
			return
		default:
			log.Printf("Unexpected message type: %T", msg)
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Not enough arguments. Usage: %s port [download-dir]\n", os.Args[0])
		os.Exit(1)
	}

	port := os.Args[1]
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalln(err.Error())
		os.Exit(1)
	}
	defer listener.Close()

	dir := "."
	if len(os.Args) >= 3 {
		dir = os.Args[2]
	}
	if err := os.Chdir(dir); err != nil {
		log.Fatalln(err)
	}

	fmt.Println("Listening on port:", port)
	fmt.Println("Download directory:", dir)
	for {
		if conn, err := listener.Accept(); err == nil {
			log.Println("Accepted connection", conn.RemoteAddr())
			handler := messages.NewMessageHandler(conn)
			go handleClient(handler)
		}
	}
}
