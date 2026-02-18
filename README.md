# file-transfer

It does work. 

What I changed:
1. Added checksum to the StorageRequest protocol buffer (Aligned with Andrew's protocol buffer).
2. Included error handling for:                                                                                                      
   - When requested file is not found                                                                                    
   - Opening, sending, and receiving files                                                                         
   - File cleanup (os.Remove()) when checksum verification fails
3. Corrected destination directory path on the client side
4. Added available disk space checking logic before transfers
5. Added filename sanitizer to prevent path traversal or malicious filenames
6. Used buffers for better file transfer performance

Improvement:

This program is not good for transferring a large file because PUT (client.go) opens the file twice. Our team decided to send the checksum at the same time when we send a storage request. But, sending a checksum verification separately is better because we can read and send the file to the network and calculate the checksum simultaneously by using io.TeeReader.

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
