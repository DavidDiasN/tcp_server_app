package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/term"
	"log"
	"net"
	"os"
	"time"
)

var (
	lastMove rune = 'w'
)

func main() {

	fmt.Println("Starting up client")
	conn, err := net.Dial("tcp", "localhost:5003")
	if err != nil {
		log.Fatal(fmt.Errorf("Ran into an error trying to connect to server: %v", err))
	}

	defer conn.Close()

	fd := int(os.Stdin.Fd())

	_, err = term.GetState(fd)
	if err != nil {
		fmt.Println(err)
		return
	}

	oldState, err := term.MakeRaw(fd)

	if err != nil {
		term.Restore(fd, oldState)
		panic(err)
	}
	defer term.Restore(fd, oldState)

	go func() {
		for {
			var boardState [][]rune
			buffer := make([]byte, 3500)

			_, err := conn.Read(buffer)
			if err != nil {
				fmt.Println("error in conn.Read")
				term.Restore(fd, oldState)
				log.Fatal(err)
			}
			//fmt.Printf("This is the raw message from conn: %v\n", string(buffer))

			serverReader := bytes.NewReader(buffer)
			decoder := json.NewDecoder(serverReader)
			updateString := ""
			if err := decoder.Decode(&boardState); err != nil {
				fmt.Printf("decoded buffer dump: %v\n", buffer)
				term.Restore(fd, oldState)
				log.Fatal(err)
			} else {
				fmt.Print("\033[2J\033[H")
				fmt.Println("\r###########################")
				for _, row := range boardState {
					updateString += "#"
					updateString += string(row)
					updateString += "#\n\r"
				}
				fmt.Printf("\r%s", updateString)
				fmt.Println("\r###########################")
			}
		}
	}()

	for {

		buff := make([]byte, 25)
		n, err := os.Stdin.Read(buff)

		if err != nil {
			fmt.Println("Error reading input:", err)
			break
		}

		var lastIndex int
		for i := 0; i < n; i++ {
			if buff[i] == 0 {
				break
			}
			lastIndex = i
		}

		if lastIndex == -1 {
			fmt.Println("Why wasn't anything read")
			continue
		}

		char := buff[lastIndex]

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
				log.Fatal(fmt.Errorf("You got an error writing to the server: %v", err))
			}

		}
		time.Sleep(300 * time.Millisecond)
		//fmt.Println("Another input")

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
