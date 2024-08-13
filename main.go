package main

import (
	"bytes"
	"embed"
	"image/color"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

//go:embed assets/*
var assets embed.FS

const (
    SCREEN_WIDTH   = 640
    SCREEN_HEIGHT  = 480
    GRID_SIZE      = 20
    VIEWPORT_WIDTH = SCREEN_WIDTH / GRID_SIZE
    TITLE          = "tr4vvyr00lz"
    POWERUP_TIME   = 300 // 5 seconds @ 60fps
)

type GameState int

const (
	StateStart GameState = iota
	StatePlaying
	StateGameOver
	StateWin
)

type Game struct {
	snake             []Point
	foods             []Point
	exit              Point
	direction         Point
	nextDirection     Point
	score             int
	state             GameState
	fontFace          *text.GoTextFace
	smallerFont       *text.GoTextFace
	maze              [][]bool
	viewportX         int
	mazeWidth         int
	mazeHeight        int
	moveCounter       int
	enemies           []Enemy
	powerUpTimer      int
	startBlinkCounter int
}

type Point struct {
	X, Y int
}

type Enemy struct {
	Position Point
}

func NewGame() *Game {
	g := &Game{
		direction:     Point{X: 1, Y: 0},
		nextDirection: Point{X: 1, Y: 0},
		state:         StateStart,
		viewportX:     0,
	}
	g.loadLevel()
	g.loadFonts()
	return g
}

func (g *Game) loadFonts() {
	fontBytes, err := assets.ReadFile("assets/pressstart2p.ttf")
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
	content, err := assets.ReadFile("assets/level.txt")
	if err != nil {
		log.Fatal(err)
	}
	levelString := string(content)

	lines := strings.Split(strings.TrimSpace(levelString), "\n")
	g.mazeHeight = len(lines)
	g.mazeWidth = len(lines[0])
	g.maze = make([][]bool, g.mazeHeight)
	g.foods = []Point{}
	g.enemies = []Enemy{}

	hasSnakeStart := false
	hasExit := false

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
				hasSnakeStart = true
			case 'E':
				g.exit = Point{X: x, Y: y}
				hasExit = true
			case 'X':
				g.enemies = append(g.enemies, Enemy{Position: Point{X: x, Y: y}})
			}
		}
	}

	if !hasSnakeStart || !hasExit || len(g.foods) == 0 {
		log.Fatal("Invalid level: missing snake start, exit, or food")
	}
}

func (g *Game) Update() error {
	switch g.state {
	case StateStart:
		if ebiten.IsKeyPressed(ebiten.KeySpace) {
			g.state = StatePlaying
		}
		// update blink counter
		g.startBlinkCounter++
		if g.startBlinkCounter >= 60 { // reset every second (60fps)
			g.startBlinkCounter = 0
		}
	case StatePlaying:
		g.handleInput()
		g.moveCounter++
		if g.moveCounter >= 10 {
			g.moveCounter = 0
			g.moveSnake()
			g.moveEnemies()
		}
		if g.powerUpTimer > 0 {
			g.powerUpTimer--
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
		g.nextDirection = Point{X: -1, Y: 0}
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) && g.direction.X == 0 {
		g.nextDirection = Point{X: 1, Y: 0}
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) && g.direction.Y == 0 {
		g.nextDirection = Point{X: 0, Y: -1}
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) && g.direction.Y == 0 {
		g.nextDirection = Point{X: 0, Y: 1}
	}
}

func (g *Game) moveSnake() {
	if g.nextDirection.X != -g.direction.X || g.nextDirection.Y != -g.direction.Y {
		g.direction = g.nextDirection
	}

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
			g.powerUpTimer = POWERUP_TIME
			break
		}
	}

	if !ateFood {
		g.snake = g.snake[:len(g.snake)-1]
	}

	g.checkEnemyCollision()
	g.adjustViewport()
}

func (g *Game) checkEnemyCollision() {
	snakeHead := g.snake[0]
	for i, enemy := range g.enemies {
		if snakeHead == enemy.Position {
			if g.powerUpTimer > 0 {
				// eat the enemy
				g.score += 5
				g.enemies = append(g.enemies[:i], g.enemies[i+1:]...)
				return
			} else {
				g.state = StateGameOver
				return
			}
		}
	}
}

func (g *Game) moveEnemies() {
	for i := range g.enemies {
		g.moveEnemy(&g.enemies[i])
	}
}

func (g *Game) moveEnemy(e *Enemy) {
	snakeHead := g.snake[0]
	possibleMoves := []Point{
		{X: 1, Y: 0}, {X: -1, Y: 0}, {X: 0, Y: 1}, {X: 0, Y: -1},
	}

	var bestMove Point
	if g.powerUpTimer > 0 {
		bestMove = g.getBestMoveAway(e.Position, snakeHead, possibleMoves)
	} else {
		bestMove = g.getBestMoveTowards(e.Position, snakeHead, possibleMoves)
	}

	newPos := Point{
		X: (e.Position.X + bestMove.X + g.mazeWidth) % g.mazeWidth,
		Y: (e.Position.Y + bestMove.Y + g.mazeHeight) % g.mazeHeight,
	}

	if !g.isCollision(newPos) {
		e.Position = newPos
	}
}

func (g *Game) getBestMoveTowards(from, to Point, moves []Point) Point {
	var bestMove Point
	minDistance := math.MaxFloat64

	for _, move := range moves {
		newPos := Point{
			X: (from.X + move.X + g.mazeWidth) % g.mazeWidth,
			Y: (from.Y + move.Y + g.mazeHeight) % g.mazeHeight,
		}

		if !g.isCollision(newPos) {
			distance := math.Hypot(float64(newPos.X-to.X), float64(newPos.Y-to.Y))
			if distance < minDistance {
				minDistance = distance
				bestMove = move
			}
		}
	}

	return bestMove
}

func (g *Game) getBestMoveAway(from, to Point, moves []Point) Point {
	var bestMove Point
	maxDistance := 0.0

	for _, move := range moves {
		newPos := Point{
			X: (from.X + move.X + g.mazeWidth) % g.mazeWidth,
			Y: (from.Y + move.Y + g.mazeHeight) % g.mazeHeight,
		}

		if !g.isCollision(newPos) {
			distance := math.Hypot(float64(newPos.X-to.X), float64(newPos.Y-to.Y))
			if distance > maxDistance {
				maxDistance = distance
				bestMove = move
			}
		}
	}

	return bestMove
}

func (g *Game) isCollision(p Point) bool {
	if g.maze[p.Y][p.X] {
		return true
	}
	for _, s := range g.snake[1:] {
		if s == p {
			return true
		}
	}
	return false
}

func (g *Game) adjustViewport() {
	snakeHead := g.snake[0]
	if snakeHead.X-g.viewportX > VIEWPORT_WIDTH*3/4 {
		g.viewportX = snakeHead.X - VIEWPORT_WIDTH*3/4
	} else if snakeHead.X-g.viewportX < VIEWPORT_WIDTH/4 {
		g.viewportX = snakeHead.X - VIEWPORT_WIDTH/4
	}

	if g.viewportX < 0 {
		g.viewportX = 0
	} else if g.viewportX > g.mazeWidth-VIEWPORT_WIDTH {
		g.viewportX = g.mazeWidth - VIEWPORT_WIDTH
	}
}

func drawMaze(g *Game, screen *ebiten.Image) {
	for y := 0; y < g.mazeHeight; y++ {
		for x := 0; x < VIEWPORT_WIDTH; x++ {
			worldX := x + g.viewportX
			if worldX < g.mazeWidth && g.maze[y][worldX] {
				vector.DrawFilledRect(screen, float32(x*GRID_SIZE), float32(y*GRID_SIZE), GRID_SIZE-1, GRID_SIZE-1, color.RGBA{100, 100, 100, 255}, true)
			}
		}
	}

	for _, food := range g.foods {
		if food.X >= g.viewportX && food.X < g.viewportX+VIEWPORT_WIDTH {
			vector.DrawFilledRect(screen, float32((food.X-g.viewportX)*GRID_SIZE), float32(food.Y*GRID_SIZE), GRID_SIZE-1, GRID_SIZE-1, color.RGBA{255, 0, 0, 255}, true)
		}
	}

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

func drawEnemies(g *Game, screen *ebiten.Image) {
	for _, enemy := range g.enemies {
		if enemy.Position.X >= g.viewportX && enemy.Position.X < g.viewportX+VIEWPORT_WIDTH {
			enemyColor := color.RGBA{255, 165, 0, 255} // orange regularly
			if g.powerUpTimer > 0 {
				enemyColor = color.RGBA{0, 0, 255, 255} // blue when vulnerable
			}
			vector.DrawFilledRect(screen, float32((enemy.Position.X-g.viewportX)*GRID_SIZE), float32(enemy.Position.Y*GRID_SIZE), GRID_SIZE-1, GRID_SIZE-1, enemyColor, true)
		}
	}
}

func drawHUD(g *Game, screen *ebiten.Image) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(10, 25)
	text.Draw(screen, "Score: "+strconv.Itoa(g.score), g.smallerFont, op)

	if g.powerUpTimer > 0 {
		powerUpText := "Power-Up: " + strconv.Itoa(g.powerUpTimer/60) // Convert frames to seconds
		op.GeoM.Translate(0, 25)
		text.Draw(screen, powerUpText, g.smallerFont, op)
	}

	if g.state == StateGameOver || g.state == StateWin {
		op := &text.DrawOptions{}

		message := "Game Over!"
		if g.state == StateWin {
			message = "You Win!"
		}
		messageWidth := float64(len(message)) * float64(g.smallerFont.Size)
		op.GeoM.Translate((float64(SCREEN_WIDTH)-messageWidth)/2, float64(SCREEN_HEIGHT)/2-25)
		text.Draw(screen, message, g.smallerFont, op)

		restartText := "Press 'R' to restart"
		restartWidth := float64(len(restartText)) * float64(g.smallerFont.Size)
		op.GeoM.Reset()
		op.GeoM.Translate((float64(SCREEN_WIDTH)-restartWidth)/2, float64(SCREEN_HEIGHT)/2+25)
		text.Draw(screen, restartText, g.smallerFont, op)
	}
}

func drawStartScreen(g *Game, screen *ebiten.Image) {
	titleWidth := float64(len(TITLE)) * float64(g.fontFace.Size)
	titleHeight := float64(g.fontFace.Size)

	op := &text.DrawOptions{}
	op.GeoM.Translate((float64(SCREEN_WIDTH)-titleWidth)/2, float64(SCREEN_HEIGHT)/2-titleHeight/2-30)
	text.Draw(screen, TITLE, g.fontFace, op)

	// only draw the start text when it should be visible
	if g.startBlinkCounter < 30 { // visible for half a second, then invisible for half a second
		startText := "Press SPACE to start"
		startWidth := float64(len(startText)) * float64(g.fontFace.Size)

		op.GeoM.Reset()
		op.GeoM.Translate((float64(SCREEN_WIDTH)-startWidth)/2, float64(SCREEN_HEIGHT)/2+30)
		text.Draw(screen, startText, g.fontFace, op)
	}
}
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 255})

	switch g.state {
	case StateStart:
		drawStartScreen(g, screen)
	case StatePlaying, StateGameOver, StateWin:
		drawMaze(g, screen)
		drawSnake(g, screen)
		drawEnemies(g, screen)
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
