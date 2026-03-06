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
	flag.StringVar(&username, "u", "", "username to login into server")
	flag.Parse()

	if len(strings.ReplaceAll(username, " ", "")) == 0 {
		fmt.Fprint(os.Stderr, "username must not be empty\n")
		os.Exit(1)
	}

	impl.DialServer(4000,
		impl.AuthCredentials{Username: username},
	)
}
