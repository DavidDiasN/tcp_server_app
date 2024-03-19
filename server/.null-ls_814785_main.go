package main

import (
	"fmt"
	"net"
)

func main() {
	ln, err := net.Listen("tcp", ":5000")
	if err != nil {
		fmt.Errorf("There was an issue making the server: %v", err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Errorf("There was an error accepting the connection: %v", err)
		}
		fmt.Println("Succesful connection")
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) error {
	for {
		buffer := make([]byte, 100)
		n, err := conn.Read(buffer)
		if err != nil {
			return fmt.Errorf("Error reading from connection:", err)
		}
		message := string(buffer[:n])
		fmt.Println(message)
	}
}
