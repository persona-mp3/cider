### Cider


Cider is an in-progress prototype of a chess game, played over the terminal.
The aim for clients to communicate over a tcp connection, which in the future
messages between them will be encrypted. And later on, use a game engine 
to render full windows instead of terminals.


### Progress
 - Server and Client initial connection [x]
 - Pairing different clients together [x]
    * Clients can pick who they want to play with aslong they're online
 - Game state []
    * Structure is defined


### Run Program
Make sure you have go installed, check [Go](https://go.dev) for installation

## 1. Clone Repo
```bash
git clone https://github.com/persona-mp3/cider.git ~/cider
cd ~/cider
```

## 2. Run Server
On a seperate terminal, you'll need to run the server
```bash
go run main.go # you should see server output and port it's listening on as of now 4000
```

## 3. Run Client
On another terminal, you can also run the client, but at the moment
you can only send messages to the server
```bash
go run internal/client/main.go # compiles all the client-files
```
You should recieve a welcome message from the server and you should also be
able to send messages to the server

To make it more interesting create as more clients as you want
