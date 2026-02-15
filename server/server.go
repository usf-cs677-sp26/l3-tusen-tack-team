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
	"syscall"
)

// hasEnoughSpace checks the server has enough space for the file
func hasEnoughSpace(size uint64) bool {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(".", &stat); err != nil {
		log.Println("Error checking disk space: ", err)
		return false
	}
	avail := stat.Bavail * uint64(stat.Bsize)
	return avail >= size
}

// sanitizeFileName removes any directory components from filename
// We want to prevent a filepath attack
func sanitizeFileName(fileName string) string {
	return filepath.Base(fileName)
}

func handleStorage(msgHandler *messages.MessageHandler, request *messages.StorageRequest) {
	fileName := sanitizeFileName(request.FileName)
	log.Println("Attempting to store", fileName)
	if !hasEnoughSpace(request.Size) {
		msgHandler.SendResponse(false, "Not enough space")
		return
	}

	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		msgHandler.SendResponse(false, err.Error())
		msgHandler.Close()
		return
	}

	msgHandler.SendResponse(true, "Ready for data")

	md5 := md5.New()
	w := io.MultiWriter(file, md5)
	_, err = io.CopyN(w, msgHandler, int64(request.Size)) /* Write and checksum as we go */
	if err != nil {
		log.Println("Error receiving file data: ", err)
		file.Close()
		os.Remove(fileName)
		msgHandler.SendResponse(false, "Failed to receive file data")
		return
	}
	file.Close()

	serverCheck := md5.Sum(nil)
	clientCheck := request.GetChecksum()

	if util.VerifyChecksum(serverCheck, clientCheck) {
		log.Println("Successfully stored file.")
		msgHandler.SendResponse(true, "Storage complete")
	} else {
		log.Println("FAILED to store file. Invalid checksum.")
		os.Remove(fileName)
		msgHandler.SendResponse(false, "Checksum verification failed")
	}
}

func handleRetrieval(msgHandler *messages.MessageHandler, request *messages.RetrievalRequest) {
	fileName := sanitizeFileName(request.FileName)
	log.Println("Attempting to retrieve", fileName)

	// Get file size and make sure it exists
	info, err := os.Stat(fileName)
	if err != nil {
		log.Println("File not found:", err)
		msgHandler.SendRetrievalResponse(false, err.Error(), 0)
		return
	}

	msgHandler.SendRetrievalResponse(true, "Ready to send", uint64(info.Size()))

	file, err := os.Open(fileName)
	if err != nil {
		log.Println("Error opening file: ", err)
		msgHandler.SendResponse(false, err.Error())
		return
	}
	md5 := md5.New()
	w := io.MultiWriter(msgHandler, md5)
	_, err = io.CopyN(w, file, info.Size()) // Checksum and transfer file at same time
	if err != nil {
		log.Println("Error sending file data: ", err)
		file.Close()
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
			log.Println("Client disconnected:", err)
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
