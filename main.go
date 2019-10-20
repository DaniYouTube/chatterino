package main

import (
	"fmt"

	"chatterino/chatterino"
)

func main() {
	srv := chatterino.NewServer()
	if err := srv.Listen("127.0.0.1:3000"); err != nil {
		fmt.Println(err)
	}
}
