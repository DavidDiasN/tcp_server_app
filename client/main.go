package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/term"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

var (
	lastMove          rune  = 'w'
	ConnectionTimeout error = errors.New("Max wait time for next packet exceeded.")
)

func main() {

	fmt.Println("Starting up client")
	conn, err := net.Dial("tcp", "localhost:5003")
	if err != nil {
		log.Fatal(fmt.Errorf("\rRan into an error trying to connect to server: %v", err))
	}

	defer deferLog()
	defer conn.Close()

	fd := int(os.Stdin.Fd())

	_, err = term.GetState(fd)
	if err != nil {
		fmt.Println(err)
		return
	}

	oldState, err := term.MakeRaw(fd)

	if err != nil {
		return
	}

	defer term.Restore(fd, oldState)
	go func() {
		for {
			timer := time.AfterFunc(3*time.Second, func() {
				gracefulClose(conn, fd, oldState, ConnectionTimeout)
			})

			var boardState [][]rune

			buffer := make([]byte, 3500)
			_, err := conn.Read(buffer)
			timer.Stop()
			if err != nil {
				fmt.Println("\rError reading from conn")
				return
			}

			if strings.Contains(string(buffer), "You Died") {
				fmt.Println("\rYou Died")
				gracefulClose(conn, fd, oldState, nil)
				return
			}

			serverReader := bytes.NewReader(buffer)
			decoder := json.NewDecoder(serverReader)

			if err := decoder.Decode(&boardState); err != nil {
				fmt.Printf("\rdecoded buffer dump: %v\n", buffer)
				return
			} else {
				displayBoard(boardState)
			}
		}
	}()

	for {

		buff := make([]byte, 25)
		n, err := os.Stdin.Read(buff)

		if err != nil {
			fmt.Println("\rError reading input:", err)
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
			fmt.Println("\rWhy wasn't anything read")
			continue
		}

		char := buff[lastIndex]

		if rune(char) == 27 {
			if err != nil {
				fmt.Println(err)
			}
			conn.Write([]byte{char})
			return
		} else if rune(char) == lastMove {
			continue
		} else {
			_, err = conn.Write([]byte{char})
			lastMove = rune(char)
			if err != nil {
				log.Fatal(fmt.Errorf("\rYou got an error writing to the server: %v", err))
			}

		}
		time.Sleep(300 * time.Millisecond)

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

func displayBoard(boardState [][]rune) {
	updateString := ""
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

func gracefulClose(conn net.Conn, fd int, oldState *term.State, err error) {
	conn.Close()
	term.Restore(fd, oldState)
	if err != nil {
		log.Fatalf("\r%v", err)
	}
}

func deferLog() {
	fmt.Println("Deferd properlly")
}
