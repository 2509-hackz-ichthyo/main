package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Piece represents a game piece with a color (0-255)
type Piece struct {
	Color uint8 // 0-255: 0-127 for black side, 128-255 for white side
}

// Square represents a square on the board that can hold at most one piece
type Square struct {
	Piece *Piece // nil if empty, otherwise contains a piece
}

// IsEmpty returns true if the square is empty
func (s *Square) IsEmpty() bool {
	return s.Piece == nil
}

// Board represents an 8x8 reversi board
type Board struct {
	Squares [8][8]Square
}

// NewBoard creates a new board with initial setup
func NewBoard() *Board {
	board := &Board{}
	
	// Initialize with starting pieces in the center
	// Black side (0-127) pieces
	board.Squares[3][3].Piece = &Piece{Color: 63}  // Black piece
	board.Squares[4][4].Piece = &Piece{Color: 63}  // Black piece
	
	// White side (128-255) pieces
	board.Squares[3][4].Piece = &Piece{Color: 191} // White piece
	board.Squares[4][3].Piece = &Piece{Color: 191} // White piece
	
	return board
}

// Game represents the game state
type Game struct {
	Board       *Board
	CurrentTurn bool   // true for black (0-127), false for white (128-255)
	NextColor   uint8  // The color of the next piece to be placed
	Rand        *rand.Rand
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrint(screen, "Hello, World!")
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 320, 240
}

// NewGame creates a new game instance
func NewGame() *Game {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	
	game := &Game{
		Board:       NewBoard(),
		CurrentTurn: true, // Start with black
		Rand:        r,
	}
	
	// Generate initial color for the first move
	game.generateNextColor()
	
	return game
}

// generateNextColor generates the next piece color based on current turn
func (g *Game) generateNextColor() {
	if g.CurrentTurn {
		// Black side: 0-127
		g.NextColor = uint8(g.Rand.Intn(128))
	} else {
		// White side: 128-255
		g.NextColor = uint8(128 + g.Rand.Intn(128))
	}
}

func main() {
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("アンミカリバーシ")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}