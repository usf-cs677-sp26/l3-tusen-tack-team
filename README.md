# file-transfer

It does work.

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
