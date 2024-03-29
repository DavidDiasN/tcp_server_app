package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	ROWS     = 25
	COLS     = 25
)

var (
	oppositeKeyDirectionMap        = map[rune]string{'w': DOWN, 's': UP, 'd': LEFT, 'a': RIGHT}
	keyDirectionMap                = map[rune]string{'w': UP, 's': DOWN, 'd': RIGHT, 'a': LEFT}
	IllegalMoveError        error  = errors.New("Illegal move entered")
	InvalidMoveError        error  = errors.New("Invalid key pressed")
	HitBounds               error  = errors.New("Hit bounds")
	UserClosedGame          error  = errors.New("User Disconnected")
	blankArr                []rune = makeEmptyArr()
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
		fmt.Printf("There was an issue making the server: %v", err)
		return
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Printf("There was an error accepting the connection: %v\n", err)
			return
		}
		fmt.Println("Succesful connection")
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) error {
	defer conn.Close()

	quit := make(chan bool)

	connectionBoard := NewBoard(ROWS, COLS, conn)

	go connectionBoard.FrameSender(quit)

	err := connectionBoard.MoveListener(quit)
	if err == UserClosedGame {
		fmt.Println("User closed the game")
		return UserClosedGame
	}

	return nil
}

type Board struct {
	rows              int
	cols              int
	boardState        [][]rune
	snakeState        []Pos
	gameConn          net.Conn
	mu                sync.Mutex
	lastInputMove     string
	lastProcessedMove string
}

func NewBoard(rows, cols int, conn net.Conn) *Board {

	var newBoardState = make([][]rune, rows)

	for i := range newBoardState {
		newBoardState[i] = make([]rune, rows)
		copy(newBoardState[i], blankArr)
	}

	startingPos := Pos{colStart, rowStart}

	return &Board{rows, cols, newBoardState, []Pos{startingPos}, conn, sync.Mutex{}, UP, UP}
}

func (b *Board) MoveListener(quit chan bool) error {

	for {
		buffer := make([]byte, 1)
		n, err := b.gameConn.Read(buffer)
		if err != nil {
			return err
		}
		if n == -1 {
			return err
		}
		char := rune(string(buffer[:n])[0])
		if validMove(char) {
			b.mu.Lock()
			b.movement(char)
			b.mu.Unlock()
		} else if char == 27 {
			b.gameConn.Close()
			quit <- true
			return UserClosedGame
		} else {
			continue
		}
	}
}

func (b *Board) FrameSender(quit chan bool) error {

	for {
		select {
		case <-quit:
			return UserClosedGame
		default:
			b.mu.Lock()
			err := b.renderBoard()
			if err != nil {
				return err
			}

			buffer := new(bytes.Buffer)
			encoder := json.NewEncoder(buffer)
			err = encoder.Encode(b.boardState)
			if err != nil {
				fmt.Printf("There was an error encoding boardState: %v\n", err)
				continue
			}
			b.gameConn.Write(buffer.Bytes())
			b.mu.Unlock()
			time.Sleep(300 * time.Millisecond)
		}
	}
}

func validMove(char rune) bool {
	return char == 'w' || char == 'a' || char == 's' || char == 'd'
}

type Pos struct {
	row int
	col int
}

func (b *Board) renderBoard() error {
	err := b.move()
	if err == IllegalMoveError {
		fmt.Printf("An Illegal move made it into move(): %v", err)
		return err
	} else if err == HitBounds {
		fmt.Println("YOU DIED")
		return err
	}
	b.updateBoard()
	//b.displayBoard()
	return nil
}

func (b *Board) updateBoard() {
	for i := range b.boardState {
		copy(b.boardState[i], blankArr)
	}
	for _, p := range b.snakeState {
		b.boardState[p.row][p.col] = 'X'
	}
}

func (b *Board) move() error {
	switch b.lastInputMove {
	case UP:
		b.lastProcessedMove = UP
		return b.moveVert(-1)
	case DOWN:
		b.lastProcessedMove = DOWN
		return b.moveVert(1)
	case LEFT:
		b.lastProcessedMove = LEFT
		return b.moveLat(-1)
	case RIGHT:
		b.lastProcessedMove = RIGHT
		return b.moveLat(1)
	default:
		return IllegalMoveError
	}
}

func (b *Board) moveVert(inc int) error {
	newPosArray := []Pos{}
	for _, p := range b.snakeState {
		if p.row+inc >= b.rows || p.row+inc <= -1 {
			return HitBounds
		}
		p = Pos{
			p.row + inc,
			p.col,
		}
		newPosArray = append(newPosArray, p)
	}
	b.snakeState = newPosArray
	return nil
}

func (b *Board) moveLat(inc int) error {
	newPosArray := []Pos{}
	for _, p := range b.snakeState {
		if p.col+inc >= b.cols || p.col+inc <= -1 {
			return HitBounds
		}
		p = Pos{
			p.row,
			p.col + inc,
		}
		newPosArray = append(newPosArray, p)
	}
	b.snakeState = newPosArray
	return nil
}

func (b *Board) movement(char rune) {
	if keyDirectionMap[char] == b.lastProcessedMove || oppositeKeyDirectionMap[char] == b.lastProcessedMove {
		return
	} else if keyDirectionMap[char] == b.lastInputMove {
		return
	}
	b.lastInputMove = keyDirectionMap[char]
}
