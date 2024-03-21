package main

import (
	"errors"
	"fmt"
	"log"
	"net"
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
	lastMove         string = UP
	IllegalMoveError error  = errors.New("Illegal move entered")
	InvalidMoveError error  = errors.New("Invalid key pressed")
	blankArr         []rune = makeEmptyArr()
)

func makeEmptyArr() []rune {
	arrRune := make([]rune, 25)
	for i := range arrRune {
		arrRune[i] = '.'
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
		}
		fmt.Println("Succesful connection")
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) error {
	rows := 25
	cols := 25

	var newBoardState = make([][]rune, rows)

	for i := range newBoardState {
		newBoardState[i] = make([]rune, rows)
		copy(newBoardState[i], blankArr)
	}

	startingPos := Pos{colStart, rowStart}
	connectionBoard := &Board{rows, cols, newBoardState, []Pos{startingPos}}

	for {
		buffer := make([]byte, 1)
		fmt.Println("We out here blocking")
		select {
		case message := <-check(conn, buffer):

			check, holdingLastMove, err := movement(connectionBoard, message, lastMove)
			if check == false {
				if err != nil {
					fmt.Println(err)
					continue
				}
				continue
			}
			lastMove = holdingLastMove
			//fmt.Println(rune(message[0]))
			connectionBoard.renderBoard()
		case <-time.After(time.Second):
			connectionBoard.renderBoard()
		default:
			connectionBoard.renderBoard()
		}
	}
}

func check(conn net.Conn, buffer []byte) chan rune {
	fmt.Println("reach?")
	res := make(chan rune)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Fatal("Oh no")
	}

	res <- rune(string(buffer[:n])[0])
	return res
}

type Board struct {
	rows       int
	cols       int
	boardState [][]rune
	snakeState []Pos
}

type Pos struct {
	row int
	col int
}

func (b *Board) renderBoard() {

	b.move(lastMove)
	b.updateBoard()
	b.displayBoard()
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
		b.moveVert(-1)
	} else if lastMove == DOWN {
		b.moveVert(1)
	} else if lastMove == LEFT {
		b.moveLat(-1)
	} else if lastMove == RIGHT {
		b.moveLat(1)
	}
}

func (b *Board) moveVert(inc int) {
	newPosArray := []Pos{}
	for _, p := range b.snakeState {
		if p.row+1 >= b.rows {
			log.Fatal("YOU DIED")
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
		if p.col+1 >= b.cols {
			log.Fatal("YOU DIED")
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
		if lastMove == DOWN {
			return false, lastMove, IllegalMoveError
		}
		lastMove = UP
		return true, lastMove, nil
	case 'd':
		if lastMove == LEFT {
			return false, lastMove, IllegalMoveError
		}
		lastMove = RIGHT
		return true, lastMove, nil
	case 'a':
		if lastMove == RIGHT {
			return false, lastMove, IllegalMoveError
		}
		lastMove = LEFT
		return true, lastMove, nil
	case 's':
		if lastMove == UP {
			return false, lastMove, IllegalMoveError
		}
		lastMove = DOWN
		return true, lastMove, nil

	default:
		return false, lastMove, InvalidMoveError
	}
}

func (b *Board) displayBoard() {
	for _, row := range b.boardState {
		fmt.Println(string(row))
	}
}
