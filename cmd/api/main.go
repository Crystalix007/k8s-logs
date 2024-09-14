package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/crystalix007/log-viewer/api"
)

func main() {
	api, err := api.New()
	if err != nil {
		panic(err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Listening on http://%s\n", listener.Addr().String())

	defer listener.Close()

	if err := http.Serve(listener, api); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}
