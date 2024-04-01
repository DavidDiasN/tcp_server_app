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
	ConnectionTimeout error    = errors.New("Max wait time for next packet exceeded.")
	blankArr          []rune   = makeEmptyArr()
	snakeState        [][2]int = [][2]int{{12, 12}}
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

	dimension := 25

	var clientBoard = make([][]rune, dimension)

	for i := range clientBoard {
		clientBoard[i] = make([]rune, dimension)
		copy(clientBoard[i], blankArr)
	}
	for _, p := range snakeState {
		clientBoard[p[0]][p[1]] = 'X'
	}

	var newPieces [][2]int
	var foodLocation = [2]int{12, 12}

	go func() {
		for {
			timer := time.AfterFunc(3*time.Second, func() {
				gracefulClose(conn, fd, oldState, ConnectionTimeout)
			})

			buffer := make([]byte, 3500)
			_, err := conn.Read(buffer)
			timer.Stop()
			if err != nil {
				fmt.Println("\rError reading from conn")
				return
			}
			//fmt.Printf("First Check length = %d\n", len(snakeState))

			if strings.Contains(string(buffer), "You Died") {
				fmt.Println("\rYou Died")
				gracefulClose(conn, fd, oldState, nil)
				return
			}

			serverReader := bytes.NewReader(buffer)
			decoder := json.NewDecoder(serverReader)

			if err := decoder.Decode(&newPieces); err != nil {
				//fmt.Printf("\rdecoded buffer dump: %v\n", buffer)
				fmt.Println("Decode error")
				return
			} else if len(newPieces) > 2 {
				clientBoard[foodLocation[0]][foodLocation[1]] = 'X'
				clientBoard[newPieces[0][0]][newPieces[0][1]] = 'O'
				foodLocation = newPieces[0]

				l := len(snakeState) - 1
				clientBoard[snakeState[l][0]][snakeState[l][1]] = ' '

				snakeState = append([][2]int{newPieces[1]}, snakeState[:l]...)
				//				fmt.Printf("Second Check length = %d\n", len(snakeState))
				snakeState = append(snakeState, newPieces[2:]...)
				//				fmt.Printf("Third Check length = %d\n", len(snakeState))
				clientBoard[snakeState[0][0]][snakeState[0][1]] = 'X'

				for i := 2; i < len(newPieces); i++ {
					clientBoard[newPieces[i][0]][newPieces[i][1]] = 'X'
				}

			} else {
				l := len(snakeState) - 1
				clientBoard[snakeState[l][0]][snakeState[l][1]] = ' '
				snakeState = append([][2]int{newPieces[0]}, snakeState[:l]...)
				clientBoard[snakeState[0][0]][snakeState[0][1]] = 'X'

			}
			displayBoard(clientBoard, len(snakeState))
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
		} else {
			_, err = conn.Write([]byte{char})

			if err != nil {
				log.Fatal(fmt.Errorf("\rYou got an error writing to the server: %v", err))
				gracefulClose(conn, fd, oldState, err)
			}

		}
		time.Sleep(70 * time.Millisecond)
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

func displayBoard(boardState [][]rune, score int) {

	updateString := ""
	fmt.Print("\033[2J\033[H")
	fmt.Println("\r###########################")
	for _, row := range boardState {
		updateString += "#"
		updateString += string(row)
		updateString += "#\n\r"
	}

	fmt.Printf("\r%s", updateString)
	fmt.Println("\r###########################\n")
	fmt.Printf("\r Score: %d\n", score)
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

func makeEmptyArr() []rune {
	arrRune := make([]rune, 25)
	for i := range arrRune {
		arrRune[i] = ' '
	}
	return arrRune
}
