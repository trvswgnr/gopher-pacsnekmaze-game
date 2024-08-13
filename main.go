package main

import (
	"bytes"
	"image/color"
	"log"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	SCREEN_WIDTH   = 640
	SCREEN_HEIGHT  = 480
	GRID_SIZE      = 20
	VIEWPORT_WIDTH = 32
	TITLE          = "tr4vvyr00lz"
)

type GameState int

const (
	StateStart GameState = iota
	StatePlaying
	StateGameOver
	StateWin
)

type Game struct {
	snake       []Point
	foods       []Point
	exit        Point
	direction   Point
	score       int
	state       GameState
	fontFace    *text.GoTextFace
	smallerFont *text.GoTextFace
	maze        [][]bool
	viewportX   int
	mazeWidth   int
	mazeHeight  int
	moveCounter int
}

type Point struct {
	X, Y int
}

func NewGame() *Game {
	g := &Game{
		direction: Point{X: 1, Y: 0},
		state:     StateStart,
		viewportX: 0,
	}
	g.loadLevel()
	g.loadFonts()
	return g
}

func (g *Game) loadFonts() {
	fontBytes, err := os.ReadFile("pressstart2p.ttf")
	if err != nil {
		log.Fatal(err)
	}

	fontFaceSource, err := text.NewGoTextFaceSource(bytes.NewReader(fontBytes))
	if err != nil {
		log.Fatal(err)
	}

	g.fontFace = &text.GoTextFace{
		Source: fontFaceSource,
		Size:   24,
	}

	g.smallerFont = &text.GoTextFace{
		Source: fontFaceSource,
		Size:   20,
	}
}

func (g *Game) loadLevel() {
	content, err := os.ReadFile("level.txt")
	if err != nil {
		log.Fatal(err)
	}
	levelString := string(content)

	lines := strings.Split(strings.TrimSpace(levelString), "\n")
	g.mazeHeight = len(lines)
	g.mazeWidth = len(lines[0])
	g.maze = make([][]bool, g.mazeHeight)
	g.foods = []Point{}

	for y, line := range lines {
		g.maze[y] = make([]bool, g.mazeWidth)
		for x, char := range line {
			switch char {
			case '#':
				g.maze[y][x] = true
			case 'F':
				g.foods = append(g.foods, Point{X: x, Y: y})
			case 'S':
				g.snake = []Point{{X: x, Y: y}}
			case 'E':
				g.exit = Point{X: x, Y: y}
			}
		}
	}
}

func (g *Game) isCollision(p Point) bool {
	if g.maze[p.Y][p.X] {
		return true
	}
	for _, s := range g.snake {
		if s == p {
			return true
		}
	}
	return false
}

func (g *Game) Update() error {
	switch g.state {
	case StateStart:
		if ebiten.IsKeyPressed(ebiten.KeySpace) {
			g.state = StatePlaying
		}
	case StatePlaying:
		g.handleInput()
		g.moveCounter++
		if g.moveCounter >= 10 {
			g.moveCounter = 0
			g.moveSnake()
		}
	case StateGameOver, StateWin:
		if ebiten.IsKeyPressed(ebiten.KeyR) {
			*g = *NewGame()
			g.state = StatePlaying
		}
	}

	return nil
}

func (g *Game) handleInput() {
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) && g.direction.X == 0 {
		g.direction = Point{X: -1, Y: 0}
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) && g.direction.X == 0 {
		g.direction = Point{X: 1, Y: 0}
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) && g.direction.Y == 0 {
		g.direction = Point{X: 0, Y: -1}
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) && g.direction.Y == 0 {
		g.direction = Point{X: 0, Y: 1}
	}
}

func (g *Game) moveSnake() {
	newHead := Point{
		X: (g.snake[0].X + g.direction.X + g.mazeWidth) % g.mazeWidth,
		Y: (g.snake[0].Y + g.direction.Y + g.mazeHeight) % g.mazeHeight,
	}

	if g.isCollision(newHead) {
		g.state = StateGameOver
		return
	}

	g.snake = append([]Point{newHead}, g.snake...)

	if newHead == g.exit {
		g.state = StateWin
		return
	}

	ateFood := false
	for i, food := range g.foods {
		if newHead == food {
			g.score++
			ateFood = true
			g.foods = append(g.foods[:i], g.foods[i+1:]...)
			break
		}
	}

	if !ateFood {
		g.snake = g.snake[:len(g.snake)-1]
	}

	if newHead.X-g.viewportX > VIEWPORT_WIDTH/2 {
		g.viewportX = newHead.X - VIEWPORT_WIDTH/2
	}
	if g.viewportX > g.mazeWidth-VIEWPORT_WIDTH {
		g.viewportX = g.mazeWidth - VIEWPORT_WIDTH
	}
}

func drawMaze(g *Game, screen *ebiten.Image) {
	// draw walls
	for y := 0; y < g.mazeHeight; y++ {
		for x := 0; x < VIEWPORT_WIDTH; x++ {
			if x+g.viewportX < g.mazeWidth && g.maze[y][x+g.viewportX] {
				vector.DrawFilledRect(screen, float32(x*GRID_SIZE), float32(y*GRID_SIZE), GRID_SIZE-1, GRID_SIZE-1, color.RGBA{100, 100, 100, 255}, true)
			}
		}
	}

	// draw food
	for _, food := range g.foods {
		if food.X >= g.viewportX && food.X < g.viewportX+VIEWPORT_WIDTH {
			vector.DrawFilledRect(screen, float32((food.X-g.viewportX)*GRID_SIZE), float32(food.Y*GRID_SIZE), GRID_SIZE-1, GRID_SIZE-1, color.RGBA{255, 0, 0, 255}, true)
		}
	}

	// draw exit
	if g.exit.X >= g.viewportX && g.exit.X < g.viewportX+VIEWPORT_WIDTH {
		vector.DrawFilledRect(screen, float32((g.exit.X-g.viewportX)*GRID_SIZE), float32(g.exit.Y*GRID_SIZE), GRID_SIZE-1, GRID_SIZE-1, color.RGBA{0, 0, 255, 255}, true)
	}
}

func drawSnake(g *Game, screen *ebiten.Image) {
	for _, p := range g.snake {
		if p.X >= g.viewportX && p.X < g.viewportX+VIEWPORT_WIDTH {
			vector.DrawFilledRect(screen, float32((p.X-g.viewportX)*GRID_SIZE), float32(p.Y*GRID_SIZE), GRID_SIZE-1, GRID_SIZE-1, color.RGBA{0, 255, 0, 255}, true)
		}
	}
}

func drawHUD(g *Game, screen *ebiten.Image) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(10, 25)
	text.Draw(screen, "Score: "+string(rune(g.score+'0')), g.smallerFont, op)

	if g.state == StateGameOver {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(SCREEN_WIDTH)/2-50, float64(SCREEN_HEIGHT)/2-25)
		text.Draw(screen, "Game Over!", g.smallerFont, op)

		op.GeoM.Reset()
		op.GeoM.Translate(float64(SCREEN_WIDTH)/2-85, float64(SCREEN_HEIGHT)/2+25)
		text.Draw(screen, "Press 'R' to restart", g.smallerFont, op)
	} else if g.state == StateWin {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(SCREEN_WIDTH)/2-40, float64(SCREEN_HEIGHT)/2-25)
		text.Draw(screen, "You Win!", g.smallerFont, op)

		op.GeoM.Reset()
		op.GeoM.Translate(float64(SCREEN_WIDTH)/2-85, float64(SCREEN_HEIGHT)/2+25)
		text.Draw(screen, "Press 'R' to restart", g.smallerFont, op)
	}
}

func drawStartScreen(g *Game, screen *ebiten.Image) {
	titleWidth := float64(len(TITLE)) * float64(g.fontFace.Size)
	titleHeight := float64(g.fontFace.Size)

	op := &text.DrawOptions{}
	op.GeoM.Translate((float64(SCREEN_WIDTH)-titleWidth)/2, float64(SCREEN_HEIGHT)/2-titleHeight/2-30)
	text.Draw(screen, TITLE, g.fontFace, op)

	startText := "Press SPACE to start"
	startWidth := float64(len(startText)) * float64(g.fontFace.Size)

	op.GeoM.Reset()
	op.GeoM.Translate((float64(SCREEN_WIDTH)-startWidth)/2, float64(SCREEN_HEIGHT)/2+30)
	text.Draw(screen, startText, g.fontFace, op)
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 255})

	switch g.state {
	case StateStart:
		drawStartScreen(g, screen)
	case StatePlaying, StateGameOver, StateWin:
		drawMaze(g, screen)
		drawSnake(g, screen)
		drawHUD(g, screen)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return SCREEN_WIDTH, SCREEN_HEIGHT
}

func main() {
	ebiten.SetWindowSize(SCREEN_WIDTH, SCREEN_HEIGHT)
	ebiten.SetWindowTitle(TITLE)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
