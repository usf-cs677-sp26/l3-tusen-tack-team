# file-transfer

## Update
I updated the following: 
- Added checksum field to StorageRequest message
- Added storage space check before accepting put request
- Error handling that happens during
    - Opening the file
    - Reading and Copying file
    - Deletion of file when the checksum verification fails

## How to Run

### Server
```bash
./bin/server <port> [download_dir]
```

### Client
#### PUT request
```bash
./bin/client <server:port> put <filename>
```
#### GET request
```bash
./bin/client <server:port> get <filename> [download_dir]
```

### Build (in case you need to rebuild)
```bash
make
```

### Clean
```bash
make clean
```

[!]`download_dir` defaults to `.` when omitted 