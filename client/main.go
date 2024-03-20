package main

import (
	"bufio"
	"fmt"
	"net"
	"os"

	"golang.org/x/term"
)

func main() {
	fd := int(os.Stdin.Fd())

	// Get the current terminal state.
	_, err := term.GetState(fd)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Put the terminal into raw mode.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	fmt.Println("Starting up client")
	conn, err := net.Dial("tcp", "localhost:5003")
	if err != nil {
		fmt.Errorf("Ran into an error trying to connect to server: %v", err)
	}
	defer conn.Close()
	reader := bufio.NewReader(os.Stdin)

	for {
		char, err := reader.ReadByte()
		if err != nil {
			fmt.Println("Error reading input:", err)
			break
		}

		_, err = conn.Write([]byte{char})

		if err != nil {
			fmt.Errorf("You got an error writing to the server: %v", err)
		}
	}
}

func scanChars(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if atEOF {
		return len(data), data, nil
	}
	return 1, data[:1], nil
}
