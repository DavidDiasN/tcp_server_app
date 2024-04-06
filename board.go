package board

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"
)

const (
	colStart       = 12
	rowStart       = 12
	snakeIncrement = 3
)

var (
	keyVectorMap             = map[rune][2]int{'w': {-1, 0}, 'd': {0, 1}, 's': {1, 0}, 'a': {0, -1}}
	keyReversalMap           = map[rune]rune{'w': 's', 's': 'w', 'd': 'a', 'a': 'd'}
	keyGrowthOptionMap       = map[rune][3]rune{'w': {'s', 'a', 'd'}, 'd': {'a', 's', 'w'}, 's': {'w', 'a', 'd'}, 'a': {'d', 's', 'w'}}
	IllegalMoveError   error = errors.New("Illegal move entered")
	InvalidMoveError   error = errors.New("Invalid key pressed")
	HitBounds          error = errors.New("Hit bounds")
	SnakeCollision     error = errors.New("Snake hit itself")
	NoValidGrowthPath  error = errors.New("No Valid growth paths")
	GameVictory        error = errors.New("Game won, game ends")
	UserClosedGame     error = errors.New("User Disconnected")
	GameQuit           error = errors.New("Game was quit")
	grewThisFrame      int   = 0
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
	lastInputMove     rune
	lastProcessedMove rune
	food              [2]int
}

func NewGame(rows, cols int, conn net.Conn) *Board {

	startingSnake := generateSnake(12, 4)

	startingFood := [2]int{0, 5}

	return &Board{rows, cols, startingSnake, conn, sync.Mutex{}, 'w', 'w', startingFood}
}

func (b *Board) MoveListener(quit chan bool) error {
	for {
		select {
		case <-quit:
			return GameQuit
		default:
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
}

func (b *Board) FrameSender(quit chan bool) error {
	grewThisFrame = snakeIncrement
	for {
		select {
		case <-quit:
			return UserClosedGame
		default:
			b.mu.Lock()
			//startTime := time.Now()
			err := b.updateSnake()
			if err == GameVictory {
				quit <- true
				return GameVictory

			}
			if err != nil {
				quit <- true
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

func generateSnake(start, size int) [][2]int {
	resSnake := [][2]int{}
	for i := 0; i < size; i++ {
		resSnake = append(resSnake, [2]int{start + i, start})
	}
	return resSnake

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
		//fmt.Println("You Died")
		b.gameConn.Write([]byte("You Died"))
		return err
	}

	if PosEqual(b.snakeState[0], b.food) {
		err = b.growSnake(snakeIncrement)
		if err != nil && len(b.snakeState) > 620 {
			b.gameConn.Write([]byte("You Won!"))
			return GameVictory
		}
		if err != nil {
			//fmt.Println("You Died")
			b.gameConn.Write([]byte("You Died"))
			return err
		}
		grewThisFrame += snakeIncrement
		landOnSnake := true
		newFoodPos := [2]int{rand.Intn(b.rows), rand.Intn(b.cols)}
		for landOnSnake {
			if collides(b.snakeState, newFoodPos) {
				grewThisFrame += snakeIncrement
				err = b.growSnake(snakeIncrement)
				if err != nil && len(b.snakeState) > 620 {
					b.gameConn.Write([]byte("You Won!"))
					return GameVictory
				}
				if err != nil {
					//fmt.Println("You Died")
					b.gameConn.Write([]byte("You Died"))
					return err
				}
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
	vector := keyVectorMap[b.lastInputMove]
	var newHead = [2]int{b.snakeState[0][0] + vector[0], b.snakeState[0][1] + vector[1]}
	if !coordsInBounds(newHead[0], b.rows) || !coordsInBounds(newHead[1], b.cols) {
		return HitBounds
	}

	if collides(b.snakeState, newHead) {
		return SnakeCollision
	}

	newPosArray := append([][2]int{newHead}, b.snakeState[:len(b.snakeState)-1]...)
	b.snakeState = newPosArray
	b.lastProcessedMove = b.lastInputMove
	return nil
}

func (b *Board) growSnake(growBy int) error {
	originalDirection := tailDirection(b.snakeState)
	newPieces, err := b.growSnakeRecurse(b.snakeState[len(b.snakeState)-1], growBy, originalDirection)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	b.snakeState = append(b.snakeState, newPieces...)

	return nil
}

func (b *Board) growSnakeRecurse(tail [2]int, growthLeft int, originalDirection rune) ([][2]int, error) {
	if growthLeft == 0 {
		return [][2]int{}, nil
	}
	directionsToCheck := keyGrowthOptionMap[originalDirection]
	var err error
	var resSnake [][2]int

	for i := 0; i < 3; i++ {
		if growthLeft == 0 {
			break
		}

		vector := keyVectorMap[directionsToCheck[i]]
		var newTail = [2]int{tail[0] + vector[0], tail[1] + vector[1]}
		if !coordsInBounds(newTail[0], b.rows) || !coordsInBounds(newTail[1], b.cols) {
			err = HitBounds
			continue
		} else if collides(b.snakeState, newTail) {
			err = SnakeCollision
			continue
		} else {
			if growthLeft-1 == 0 {
				response := [][2]int{newTail}
				return response, nil
			} else {
				rest, recursErr := b.growSnakeRecurse(newTail, growthLeft-1, keyReversalMap[directionsToCheck[i]])
				growthLeft--

				if recursErr == NoValidGrowthPath {
					growthLeft++
					continue
				} else if recursErr != nil {
					growthLeft++
					continue
				}
				resSnake = append([][2]int{newTail}, rest...)
				break
			}
		}
	}

	if len(resSnake) == 3 {

		return resSnake, nil
	}

	return resSnake, err
}

func collides(snake [][2]int, newPos [2]int) bool {
	for _, p := range snake {
		if PosEqual(p, newPos) {
			return true
		}
	}
	return false
}

func coordsInBounds(x, upperLimit int) bool {
	return x < upperLimit && x > -1
}

func (b *Board) movement(char rune) {
	if char == b.lastProcessedMove || keyReversalMap[char] == b.lastProcessedMove {
		return
	} else if char == b.lastInputMove {
		return
	}

	b.lastInputMove = char
}

func tailDirection(snakeState [][2]int) rune {
	l := len(snakeState) - 1
	last := snakeState[l]
	second2Last := snakeState[l-1]
	if last[0] == second2Last[0] {
		if last[1] > second2Last[1] {
			return 'a'
		} else {
			return 'd'
		}
	} else {
		if last[0] > second2Last[0] {
			return 'w'
		} else {
			return 's'
		}
	}
}
