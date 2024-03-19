package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	fmt.Println("Starting up client")
	conn, err := net.Dial("tcp", "localhost:5000")
	if err != nil {
		fmt.Errorf("Ran into an error trying to connect to server: %v", err)
	}
	defer conn.Close()
	scanner := bufio.NewScanner(os.Stdin)

	for {

		scanner.Scan()
		str := string(scanner.Bytes())
		if str == "" {
			continue
		}
		byteArray := []byte(str)
		_, err = conn.Write(byteArray)

		if err != nil {
			fmt.Errorf("You got an error writing to the server: %v", err)
		}
	}
}
