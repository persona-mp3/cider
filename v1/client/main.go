package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/persona-mp3/internal/client/impl"
)

func main() {
	var username string
	var secure bool
	var at string
	flag.BoolVar(&secure, "secure", false, "Communicate with server tls")
	flag.StringVar(&username, "u", "", "username to get authenticated by server")
	// for dev purposes, we should stick to IP address
	// but later on, the default will be cidervine.com
	flag.StringVar(&at, "at", "localhost:4000", "Address of the server, IP.ADDRESS")
	flag.Parse()

	if len(strings.ReplaceAll(username, " ", "")) == 0 {
		fmt.Fprint(os.Stderr, "username must not be empty\n")
		os.Exit(1)
	}
	creds := impl.AuthCredentials{
		Username: username,
	}

	impl.MainDialer(at, creds, secure)
}
