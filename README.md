# file-transfer

It does work. But, this is not good for transferring a large file because PUT (client.go) opens the file twice. Our team decided to send the checksum at the same time when we send a storage request. But, sending a checksum verification separately is better because we can read and send the file to the network and calculate the checksum simultaneously by using io.TeeReader.

## How to run

# Server
```
./bin/server 8080 downloads/
```

# Client
Wants to send a file to the server
```
./bin/client localhost:8080 put Recipes.pdf
```

Wants to fetch a file from the server
```
./bin/client localhost:8080 get Recipes.pdf client-folder/
```
