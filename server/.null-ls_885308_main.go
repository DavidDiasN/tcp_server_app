package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

const (
	UP       = "UP"
	DOWN     = "DOWN"
	LEFT     = "LEFT"
	RIGHT    = "RIGHT"
	colStart = 12
	rowStart = 12
)

var (
	oppositeKeyDirectionMap        = map[rune]string{'w': DOWN, 's': UP, 'd': LEFT, 'a': RIGHT}
	keyDirectionMap                = map[rune]string{'w': UP, 's': DOWN, 'd': RIGHT, 'a': LEFT}
	lastInputMove           string = UP
	lastProcessedMove       string = UP
	IllegalMoveError        error  = errors.New("Illegal move entered")
	InvalidMoveError        error  = errors.New("Invalid key pressed")
	blankArr                []rune = makeEmptyArr()
	takeABreak              bool   = false
)

func makeEmptyArr() []rune {
	arrRune := make([]rune, 25)
	for i := range arrRune {
		arrRune[i] = ' '
	}
	return arrRune
}

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
      return
		}
		fmt.Println("Succesful connection")
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) error {
	defer conn.Close()
	rows := 25
	cols := 25

	var newBoardState = make([][]rune, rows)

	for i := range newBoardState {
		newBoardState[i] = make([]rune, rows)
		copy(newBoardState[i], blankArr)
	}

	startingPos := Pos{colStart, rowStart}

	connectionBoard := &Board{rows, cols, newBoardState, []Pos{startingPos}, sync.Mutex{}}

	message := make(chan rune)
	go func() {
		for {
			buffer := make([]byte, 1)
			n, err := conn.Read(buffer)
			if err != nil {
				return
			}
			if n == -1 {
				return
			}
			message <- rune(string(buffer[:n])[0])
			time.Sleep(300 * time.Millisecond)
		}

	}()

	go func() {
		for {
			connectionBoard.mu.Lock()
			connectionBoard.renderBoard()
			buffer := new(bytes.Buffer)
			encoder := json.NewEncoder(buffer)
			err := encoder.Encode(connectionBoard.boardState)
			if err != nil {
				log.Fatal(err)
			}
			conn.Write(buffer.Bytes())
			connectionBoard.mu.Unlock()
			time.Sleep(300 * time.Millisecond)
		}
	}()

	for {
		select {
		case char := <-message:
			if keyDirectionMap[char] == lastProcessedMove {
				continue
			} else if oppositeKeyDirectionMap[char] == lastProcessedMove {
				continue
			}
			connectionBoard.mu.Lock()
			check, holdingLastMove, err := movement(connectionBoard, char, lastInputMove)
			if check == false {
				if err != nil {
					fmt.Println(err)
				}
			}
			lastInputMove = holdingLastMove
			connectionBoard.mu.Unlock()

		case <-time.After(250 * time.Millisecond):
			continue
		}
	}
}

type Board struct {
	rows       int
	cols       int
	boardState [][]rune
	snakeState []Pos
	mu         sync.Mutex
}

type Pos struct {
	row int
	col int
}

func (b *Board) renderBoard() {
	b.move(lastInputMove)
	b.updateBoard()
	//b.displayBoard()
}

func (b *Board) updateBoard() {
	for i := range b.boardState {
		copy(b.boardState[i], blankArr)
	}
	for _, p := range b.snakeState {
		b.boardState[p.row][p.col] = 'X'
	}
}

func (b *Board) move(lastMove string) {

	if lastMove == UP {
		lastProcessedMove = UP
		b.moveVert(-1)
	} else if lastMove == DOWN {
		lastProcessedMove = DOWN
		b.moveVert(1)
	} else if lastMove == LEFT {
		lastProcessedMove = LEFT
		b.moveLat(-1)
	} else if lastMove == RIGHT {
		lastProcessedMove = RIGHT
		b.moveLat(1)
	}
}

func (b *Board) moveVert(inc int) {
	newPosArray := []Pos{}
	for _, p := range b.snakeState {
		if p.row+inc >= b.rows || p.row+inc <= -1 {
			log.Fatal("You died")
		}
		p = Pos{
			p.row + inc,
			p.col,
		}
		newPosArray = append(newPosArray, p)
	}
	b.snakeState = newPosArray
}

func (b *Board) moveLat(inc int) {
	newPosArray := []Pos{}
	for _, p := range b.snakeState {
		if p.col+inc >= b.cols || p.col+inc <= -1 {
			log.Fatal("You died")
		}
		p = Pos{
			p.row,
			p.col + inc,
		}
		newPosArray = append(newPosArray, p)
	}
	b.snakeState = newPosArray
}

func movement(board *Board, message rune, lastMove string) (bool, string, error) {
	switch message {
	case 'w':
		lastMove = UP
		return true, lastMove, nil
	case 'd':
		lastMove = RIGHT
		return true, lastMove, nil
	case 'a':
		lastMove = LEFT
		return true, lastMove, nil
	case 's':
		lastMove = DOWN
		return true, lastMove, nil

	default:
		return false, lastMove, InvalidMoveError
	}
}

//func (b *Board) displayBoard() {
//	for _, row := range b.boardState {
//		fmt.Println(string(row))
//	}
//}
