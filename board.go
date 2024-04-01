package board

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
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
	oppositeKeyDirectionMap       = map[rune]string{'w': DOWN, 's': UP, 'd': LEFT, 'a': RIGHT}
	keyDirectionMap               = map[rune]string{'w': UP, 's': DOWN, 'd': RIGHT, 'a': LEFT}
	IllegalMoveError        error = errors.New("Illegal move entered")
	InvalidMoveError        error = errors.New("Invalid key pressed")
	HitBounds               error = errors.New("Hit bounds")
	SnakeCollision          error = errors.New("Snake hit itself")
	UserClosedGame          error = errors.New("User Disconnected")
	grewThisFrame           int   = 0
	snakeIncrement          int   = 3
)

type Connection interface {
	io.Reader
	io.Writer
	io.Closer
}

type Board struct {
	rows              int
	cols              int
	snakeState        [][2]int
	gameConn          Connection
	mu                sync.Mutex
	lastInputMove     string
	lastProcessedMove string
	food              [2]int
}

func NewGame(rows, cols int, conn net.Conn) *Board {

	startingSnake := generateSnake(12, 4)

	startingFood := [2]int{rand.Intn(rows), rand.Intn(cols)}

	return &Board{rows, cols, startingSnake, conn, sync.Mutex{}, UP, UP, startingFood}
}

func (b *Board) MoveListener(quit chan bool) error {
	for {
		buffer := make([]byte, 1)
		//timer := time.Now()
		n, err := b.gameConn.Read(buffer)
		//fmt.Printf("time to read: %v \n", time.Since(timer).Seconds())
		if err != nil {
			fmt.Println("Read error")
			return err
		}
		if n == -1 {
			fmt.Println("-1 error")
			return err
		}
		char := rune(string(buffer[:n])[0])
		//fmt.Println(char)
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
		time.Sleep(17 * time.Millisecond)
	}
}

func generateSnake(start, size int) [][2]int {
	resSnake := [][2]int{}
	for i := 0; i < size; i++ {
		resSnake = append(resSnake, [2]int{start + i, start})
	}
	return resSnake

}

func (b *Board) FrameSender(quit chan bool) error {
	grewThisFrame = 3
	for {
		select {
		case <-quit:
			return UserClosedGame
		default:
			b.mu.Lock()
			//startTime := time.Now()
			err := b.updateSnake()
			if err != nil {
				return err
			}

			buffer := new(bytes.Buffer)
			encoder := json.NewEncoder(buffer)
			if grewThisFrame != 0 {
				newPieces := [][2]int{}
				for i := len(b.snakeState) - grewThisFrame; i < len(b.snakeState); i++ {
					newPieces = append(newPieces, b.snakeState[i])
				}
				newPieces = append([][2]int{b.food, b.snakeState[0]}, newPieces...)
				//fmt.Println(newPieces)
				err = encoder.Encode(newPieces)
				grewThisFrame = 0
			} else {
				err = encoder.Encode([][2]int{b.snakeState[0]})
			}
			if err != nil {
				fmt.Printf("There was an error encoding boardState: %v\n", err)
				continue
			}
			b.gameConn.Write(buffer.Bytes())

			b.mu.Unlock()
			//fmt.Printf("Snake was locked for %v seconds\n", time.Since(startTime).Seconds())
			time.Sleep(150 * time.Millisecond)
		}
	}
}

func validMove(char rune) bool {
	return char == 'w' || char == 'a' || char == 's' || char == 'd'
}

func PosEqual(a, b [2]int) bool {
	return a[0] == b[0] && a[1] == b[1]
}

func (b *Board) updateSnake() error {
	err := b.move()
	if err == IllegalMoveError {
		fmt.Printf("An Illegal move made it into move(): %v", err)
	} else if err == HitBounds || err == SnakeCollision {
		fmt.Println("You Died")
		b.gameConn.Write([]byte("You Died"))
		return err
	}

	if PosEqual(b.snakeState[0], b.food) {
		b.growSnake(snakeIncrement)
		grewThisFrame += snakeIncrement
		landOnSnake := true
		newFoodPos := [2]int{rand.Intn(b.rows), rand.Intn(b.cols)}
		for landOnSnake {
			if collides(b.snakeState, newFoodPos) {
				grewThisFrame += snakeIncrement
				b.growSnake(snakeIncrement)
				newFoodPos = [2]int{rand.Intn(b.rows), rand.Intn(b.cols)}
			} else {
				landOnSnake = false
				b.food = newFoodPos
				return nil
			}
		}
	}

	return nil
}

func (b *Board) move() error {
	switch b.lastInputMove {
	case UP:
		b.lastProcessedMove = UP
		return b.processMove(0, -1)
	case DOWN:
		b.lastProcessedMove = DOWN
		return b.processMove(0, 1)
	case LEFT:
		b.lastProcessedMove = LEFT
		return b.processMove(1, -1)
	case RIGHT:
		b.lastProcessedMove = RIGHT
		return b.processMove(1, 1)
	default:
		return IllegalMoveError
	}
}

func (b *Board) processMove(pos, inc int) error {

	if !coordsInBounds(b.snakeState[0][pos] + inc) {
		return HitBounds
	}
	var newHead [2]int
	if pos == 1 {
		newHead = [2]int{b.snakeState[0][0], b.snakeState[0][1] + inc}
	} else {
		newHead = [2]int{b.snakeState[0][0] + inc, b.snakeState[0][1]}
	}

	if collides(b.snakeState, newHead) {
		return SnakeCollision
	}
	newPosArray := append([][2]int{newHead}, b.snakeState[:len(b.snakeState)-1]...)

	b.snakeState = newPosArray
	return nil

}

func (b *Board) growSnake(growBy, pos, inc int) error {
	l := len(b.snakeState) - 1
// Okay I think my idea is to just grow in the opposite direction that you are currently going in and making this whole thing recursive.
// You can't grow in the direction you previously grew in so that part makes sense but the only problem with this method is
// I need to be able to try a new root if I hit an issue before the snake is done growing. That also means changes have to be done on the final
// iteration
	if oppositeKeyDirectionMap[(b.snakeState)] == 
	
	i := 0
	for i < inc {
		x := b.snakeState[l][0]
		y := b.snakeState[l][1]
		if coordsInBounds(x + 1) {
			if collides(b.snakeState, [2]int{x + 1, y}) {
			}
			b.snakeState = append(b.snakeState, [2]int{x + 1, y})
			i++
		} else if coordsInBounds(x - 1) {
			if collides(b.snakeState, [2]int{x - 1, y}) {

			}

			b.snakeState = append(b.snakeState, [2]int{x - 1, y})
			i++
		} else if coordsInBounds(y + 1) {
			if collides(b.snakeState, [2]int{x, y + 1}) {

			}

			b.snakeState = append(b.snakeState, [2]int{x, y + 1})
			i++
		} else if coordsInBounds(y - 1) {
			if collides(b.snakeState, [2]int{x, y - 1}) {

			}

			b.snakeState = append(b.snakeState, [2]int{x, y - 1})
			i++
		}

	}
	return nil
}

func collides(snake [][2]int, newPos [2]int) bool {
	for _, p := range snake {
		if PosEqual(p, newPos) {
			return true
		}
	}
	return false
}

func coordsInBounds(x int) bool {
	return x < 25 && x > -1
}

func (b *Board) movement(char rune) {
	if keyDirectionMap[char] == b.lastProcessedMove || oppositeKeyDirectionMap[char] == b.lastProcessedMove {
		return
	} else if keyDirectionMap[char] == b.lastInputMove {
		return
	}
	b.lastInputMove = keyDirectionMap[char]
}

func tailDirection(snakeState [][2]int) string {
	l := len(snakeState) - 1
	last := snakeState[l]
	second2Last := snakeState[l-1]
	if last[0] == second2Last[0] {
		if last[1] > second2Last[1] {
			return DOWN
		} else {
			return UP
		}
	} else {
		if last[0] > second2Last[0] {
			return LEFT
		} else {
			return RIGHT
		}
	}
}
