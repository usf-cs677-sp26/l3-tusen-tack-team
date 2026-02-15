package util

import (
	"log"
	"reflect"

	"golang.org/x/sys/unix"
)

func VerifyChecksum(serverCheck []byte, clientCheck []byte) bool {
	log.Printf("Server checksum: %x\n", serverCheck)
	log.Printf("Client checksum: %x\n", clientCheck)
	if reflect.DeepEqual(clientCheck, serverCheck) {
		log.Println("Checksums match")
		return true
	} else {
		log.Println("Checksums DO NOT match")
		return false
	}
}

func GetStorageSize(path string) (uint64, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, err
	}
	return st.Bavail * uint64(st.Bsize), nil
}
