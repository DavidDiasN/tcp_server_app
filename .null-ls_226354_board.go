package board

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
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



