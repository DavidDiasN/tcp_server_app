package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
	fmt.Println("This is my client program")
	conn, err := net.Dial("tcp", "localhost:5000")
	if err != nil {
		fmt.Errorf("Ran into an error trying to connect to server: %v", err)
	}
	defer conn.Close()

	for i := 0; i < 15; i++ {
		time.Sleep(1 * time.Millisecond)
		str := fmt.Sprintf("This is warning %d", i)
		byteArray := []byte(str)
		_, err = conn.Write(byteArray)

		if err != nil {
			fmt.Errorf("You got an error writing to the server: %v", err)
		}
	}
}
