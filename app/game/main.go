package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
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

const (
	BoardSize   = 8
	CellSize    = 60
	BoardOffset = 50
)

func (g *Game) Update() error {
	// Handle mouse clicks
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		
		// Convert screen coordinates to board coordinates
		boardX := (mx - BoardOffset) / CellSize
		boardY := (my - BoardOffset) / CellSize
		
		// Check if click is within board bounds and place piece
		if boardX >= 0 && boardX < BoardSize && boardY >= 0 && boardY < BoardSize {
			g.placePiece(boardX, boardY)
		}
	}
	
	return nil
}

// colorToRGB converts a 0-255 color value to RGB
func colorToRGB(c uint8) color.RGBA {
	// Simple color mapping: use the value for green channel, 
	// and create contrast between black/white sides
	if c < 128 {
		// Black side: darker colors
		return color.RGBA{R: c / 2, G: c, B: c / 2, A: 255}
	} else {
		// White side: brighter colors
		adjusted := c - 128
		return color.RGBA{R: 128 + adjusted/2, G: 255, B: 128 + adjusted/2, A: 255}
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Clear screen with light gray
	screen.Fill(color.RGBA{240, 240, 240, 255})
	
	// Draw board grid
	for i := 0; i <= BoardSize; i++ {
		x := float32(BoardOffset + i*CellSize)
		y1 := float32(BoardOffset)
		y2 := float32(BoardOffset + BoardSize*CellSize)
		
		// Vertical lines
		vector.StrokeLine(screen, x, y1, x, y2, 2, color.Black, false)
		
		// Horizontal lines
		y := float32(BoardOffset + i*CellSize)
		x1 := float32(BoardOffset)
		x2 := float32(BoardOffset + BoardSize*CellSize)
		vector.StrokeLine(screen, x1, y, x2, y, 2, color.Black, false)
	}
	
	// Draw pieces
	for x := 0; x < BoardSize; x++ {
		for y := 0; y < BoardSize; y++ {
			if !g.Board.Squares[x][y].IsEmpty() {
				piece := g.Board.Squares[x][y].Piece
				pieceColor := colorToRGB(piece.Color)
				
				centerX := float32(BoardOffset + x*CellSize + CellSize/2)
				centerY := float32(BoardOffset + y*CellSize + CellSize/2)
				radius := float32(CellSize/2 - 4)
				
				vector.DrawFilledCircle(screen, centerX, centerY, radius, pieceColor, false)
				vector.StrokeCircle(screen, centerX, centerY, radius, 2, color.Black, false)
			}
		}
	}
	
	// Draw UI information
	currentPlayer := g.getCurrentPlayerSide()
	nextColorRGB := colorToRGB(g.NextColor)
	
	infoText := fmt.Sprintf("Current Player: %s\nNext Color: %d\nClick to place piece", 
		currentPlayer, g.NextColor)
	ebitenutil.DebugPrint(screen, infoText)
	
	// Draw next color preview
	previewX := float32(BoardOffset + BoardSize*CellSize + 20)
	previewY := float32(BoardOffset + 60)
	vector.DrawFilledCircle(screen, previewX, previewY, 20, nextColorRGB, false)
	vector.StrokeCircle(screen, previewX, previewY, 20, 2, color.Black, false)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 800, 600
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

// isBlackSide returns true if the color belongs to black side (0-127)
func isBlackSide(color uint8) bool {
	return color < 128
}

// isWhiteSide returns true if the color belongs to white side (128-255)
func isWhiteSide(color uint8) bool {
	return color >= 128
}

// switchTurn switches the current turn and generates next color
func (g *Game) switchTurn() {
	g.CurrentTurn = !g.CurrentTurn
	g.generateNextColor()
}

// getCurrentPlayerSide returns the current player's side as string
func (g *Game) getCurrentPlayerSide() string {
	if g.CurrentTurn {
		return "Black"
	}
	return "White"
}

// Direction represents the 8 directions on the board
type Direction struct {
	dx, dy int
}

var directions = []Direction{
	{-1, -1}, {-1, 0}, {-1, 1},
	{0, -1},           {0, 1},
	{1, -1}, {1, 0}, {1, 1},
}

// isValidPosition checks if the position is within board bounds
func isValidPosition(x, y int) bool {
	return x >= 0 && x < 8 && y >= 0 && y < 8
}

// findFlankingPieces finds pieces that would be flanked by placing a piece at (x, y)
func (g *Game) findFlankingPieces(x, y int, color uint8) [][]Position {
	if !g.Board.Squares[x][y].IsEmpty() {
		return nil
	}
	
	var flankingLines [][]Position
	currentSide := isBlackSide(color)
	
	for _, dir := range directions {
		var line []Position
		nx, ny := x+dir.dx, y+dir.dy
		
		// Look for opposite side pieces
		for isValidPosition(nx, ny) && !g.Board.Squares[nx][ny].IsEmpty() {
			piece := g.Board.Squares[nx][ny].Piece
			if isBlackSide(piece.Color) == currentSide {
				// Found same side piece, this line is valid if we have pieces to flank
				if len(line) > 0 {
					flankingLines = append(flankingLines, line)
				}
				break
			} else {
				// Opposite side piece, add to potential flanking line
				line = append(line, Position{nx, ny})
			}
			nx, ny = nx+dir.dx, ny+dir.dy
		}
	}
	
	return flankingLines
}

// Position represents a position on the board
type Position struct {
	X, Y int
}

// isValidMove checks if placing a piece at (x, y) is a valid move
func (g *Game) isValidMove(x, y int) bool {
	if !isValidPosition(x, y) || !g.Board.Squares[x][y].IsEmpty() {
		return false
	}
	
	flankingLines := g.findFlankingPieces(x, y, g.NextColor)
	return len(flankingLines) > 0
}

// placePiece places a piece at (x, y) and applies color changes according to the rules
func (g *Game) placePiece(x, y int) bool {
	if !g.isValidMove(x, y) {
		return false
	}
	
	// Place the new piece
	g.Board.Squares[x][y].Piece = &Piece{Color: g.NextColor}
	
	// Find all flanking lines
	flankingLines := g.findFlankingPieces(x, y, g.NextColor)
	
	// Process each flanking line
	for _, line := range flankingLines {
		if len(line) > 0 {
		// Get the flanking pieces
		end := line[len(line)-1]
		
		a1 := g.NextColor // The newly placed piece
		a2 := g.Board.Squares[end.X][end.Y].Piece.Color // The far flanking piece			// Apply color change to all pieces in between
			for _, pos := range line {
				piece := g.Board.Squares[pos.X][pos.Y].Piece
				b1 := piece.Color
				
				// Apply the color change formula: c = (a1 + a2 + b1) / 3
				newColor := uint8((uint16(a1) + uint16(a2) + uint16(b1)) / 3)
				piece.Color = newColor
			}
		}
	}
	
	// Switch turn and generate next color
	g.switchTurn()
	return true
}

func main() {
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("アンミカリバーシ")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}