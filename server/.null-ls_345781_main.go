package main

import (
	"fmt"
	"net"
)

const (
	UP          = "UP"
	DOWN        = "DOWN"
	LEFT        = "LEFT"
	RIGHT       = "RIGHT"
	ILLEGALMOVE = false
)

var lastMove string

func main() {
	ln, err := net.Listen("tcp", ":5003")
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
		switch rune(message[0]) {
		case 'w':
			if lastMove == DOWN {
				fmt.Println(ILLEGALMOVE)
				break
			}
			fmt.Println(UP)
			lastMove = UP
		case 'd':
			if lastMove == LEFT {
				fmt.Println(ILLEGALMOVE)
				break
			}
			fmt.Println(RIGHT)
			lastMove = RIGHT
		case 'a':
			if lastMove == RIGHT {
				fmt.Println(ILLEGALMOVE)
				break
			}
			fmt.Println(LEFT)
			lastMove = LEFT
		case 's':
			if lastMove == UP {
				fmt.Println(ILLEGALMOVE)
				break
			}
			fmt.Println(DOWN)
			lastMove = DOWN
		}
		//if len(message) > 1 {
		//	fmt.Println("Double send bad, well not really bad but I need to implement a buffer or something to deal with this")
		//	continue
		//}
		fmt.Println(rune(message[0]))
	}
}
