package board

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	IllegalMoveError        error  = errors.New("Illegal move entered")
	InvalidMoveError        error  = errors.New("Invalid key pressed")
	HitBounds               error  = errors.New("Hit bounds")
	UserClosedGame          error  = errors.New("User Disconnected")
	blankArr                []rune = makeEmptyArr()
)

type Connection interface {
	io.Reader
	io.Writer
	io.Closer
}

func makeEmptyArr() []rune {
	arrRune := make([]rune, 25)
	for i := range arrRune {
		arrRune[i] = ' '
	}
	return arrRune
}

type Board struct {
	rows              int
	cols              int
	boardState        [][]rune
	snakeState        []Pos
	gameConn          Connection
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

	startingSnake := []Pos{Pos{colStart, rowStart}, Pos{colStart - 1, rowStart - 1}, Pos{colStart - 2, rowStart - 2}, Pos{colStart - 3, rowStart - 3}}

	return &Board{rows, cols, newBoardState, startingSnake, conn, sync.Mutex{}, UP, UP}
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
	} else if err == HitBounds {
		fmt.Println("You Died")
		b.gameConn.Write([]byte("You Died"))
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
	if b.snakeState[0].row+inc >= b.rows || b.snakeState[0].row+inc <= -1 {
		return HitBounds
	}
	newHead := Pos{b.snakeState[0].row, b.snakeState[0].col + inc}

	newPosArray := append([]Pos{newHead}, b.snakeState[:len(b.snakeState)-2]...)

	b.snakeState = newPosArray
	return nil
}

func (b *Board) moveLat(inc int) error {

	if b.snakeState[0].row+inc >= b.rows || b.snakeState[0].col+inc <= -1 {
		return HitBounds
	}
	newHead := Pos{b.snakeState[0].row + inc, b.snakeState[0].col}

	newPosArray := append([]Pos{newHead}, b.snakeState[:len(b.snakeState)-2]...)

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
