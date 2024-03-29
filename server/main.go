package main

import (
	"fmt"
	"github.com/DavidDiasN/tcp_server_app"
	"net"
)

func main() {
	ln, err := net.Listen("tcp", ":5003")
	if err != nil {
		fmt.Printf("There was an issue making the server: %v", err)
		return
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Printf("There was an error accepting the connection: %v\n", err)
			return
		}
		fmt.Println("Succesful connection")
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) error {
	defer conn.Close()

	quit := make(chan bool)

	connectionBoard := board.NewBoard(25, 25, conn)
	// add more channels to catch errors
	go connectionBoard.FrameSender(quit)

	err := connectionBoard.MoveListener(quit)
	if err == board.UserClosedGame {
		fmt.Println("User closed the game")
		return board.UserClosedGame
	}

	return nil
}
