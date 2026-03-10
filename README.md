### Cider


Cider is an in-progress prototype of a chess game, played over the terminal.
The aim for clients to communicate over a tcp connection, which in the future
messages between them will be encrypted. And later on, use a game engine 
to render full windows instead of terminals.


### Progress
 - Server and Client initial connection [x]
 - Pairing different clients together [x]
    * Clients can pick who they want to play with aslong they're online {x}
 - Game state []
    * Structure is defined
 - Game Packets are sent over TLS [x]
 


### Requirements
1. [Go](https://go.dev)

2. [Postgresql](https://postgresql.org), preferrably version 15


## 1. Clone Repo
```bash
git clone https://github.com/persona-mp3/cider.git ~/cider
cd ~/cider
```


### Run Program
1. If you want to run the program without any special configurations, 
you can run both the server and client in default mode. But you 
should fist run the `setup.sh` script first, as it initialises the database
and other environmental variables for you.

## To run the server in default mode, (this is not done over any encryption)
```bash
    go run main.go -port 4000 # that will start a tcp server and listen for incoming connections


    # You must also run the client in default mode
    # it connnects to localhost:4000 automatically,  but you must provide a username from the db
    go run internal/client/main.go -u "persona" 
```


## To run the server over TLS, please see the `gen-tls-cert.sh` script to 
do this for you. 
```bash
    chmod u+x ./gen-tls-cert.sh # allows it to be executable

    ./gen-tls-cert.sh # execute the script and feel free to play with it

    go run main.go -port 1738 secure=true # this will start the server to accept connections over tls


    # for client
    go run internal/client/main.go -at localhost:1738 -secure=true # now the client will also run over tls
```
### Important: Subject to change
The client implementation of TLS is temporary for now. Later on, 
TLS will be set by default when connecting to the main server, You should only use 
`default` mode for testing or development. 


## Operating the client
At the moment, the connection between any two conneted clients can be set up..
After running the client, you should see a list of `active` friends. And you'd 
most likely need to spin another client in another terminal.

There's a new change to implement an actual UI but this is how you can initiate 
a new game with a connected user
```bash
ng morty* I challenge you!
```

If `morty` is online, he get's the challenge request and both 
clients can play. Right now, the play mechanism is just sending strings over. 
Timeouts do happend, 8s being IDLE, and the turn is handed over to the other player!


To make it more interesting create as more clients as you want aslong as they 
are registered in the database!
