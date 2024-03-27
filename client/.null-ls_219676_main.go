package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/term"
	"log"
	"net"
	"os"
  tea "github.com/charmbracelet/bubbletea"
)

var (
	lastMove rune = 'w'
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
		log.Fatal(fmt.Errorf("Ran into an error trying to connect to server: %v", err))
	}
	defer conn.Close()
	reader := bufio.NewReader(os.Stdin)

	go func() {
		for {
			var boardState [][]rune
			buffer := make([]byte, 3500)

			_, err := conn.Read(buffer)
			if err != nil {
				fmt.Println("error in conn.Read")
				term.Restore(int(os.Stdin.Fd()), oldState)
				log.Fatal(err)
			}
			//fmt.Printf("This is the raw message from conn: %v\n", string(buffer))

			serverReader := bytes.NewReader(buffer)
			decoder := json.NewDecoder(serverReader)
			updateString := ""
			if err := decoder.Decode(&boardState); err != nil {
				fmt.Printf("decoded buffer dump: %v\n", buffer)
				term.Restore(int(os.Stdin.Fd()), oldState)
				log.Fatal(err)
			} else {
				for _, row := range boardState {
					updateString += string(row)
				}
				fmt.Println(updateString)
			}
		}
	}()

	for {
		char, err := reader.ReadByte()
		if err != nil {
			fmt.Println("Error reading input:", err)
			break
		}

		if rune(char) == 27 {
			if err != nil {
				fmt.Println(err)
			}
			return
		} else if rune(char) == lastMove {
			continue
		} else {
			_, err = conn.Write([]byte{char})
			lastMove = rune(char)
			if err != nil {
				fmt.Errorf("You got an error writing to the server: %v", err)
			}

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
