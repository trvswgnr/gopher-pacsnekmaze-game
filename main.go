package main

import (
	"image/color"
	"log"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
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
	snake      []Point
	foods      []Point
	exit       Point
	direction  Point
	score      int
	state      GameState
	font       font.Face
	maze       [][]bool
	viewportX  int
	mazeWidth  int
	mazeHeight int
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
	g.loadFont()
	return g
}

func (g *Game) loadFont() {
	tt, err := opentype.Parse(goregular.TTF)
	if err != nil {
		log.Fatal(err)
	}
	g.font, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    24,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
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
		moved := g.handleInput()
		if moved {
			newHead := Point{
				X: g.snake[0].X + g.direction.X,
				Y: g.snake[0].Y + g.direction.Y,
			}

			if newHead.X < 0 || newHead.X >= g.mazeWidth || newHead.Y < 0 || newHead.Y >= g.mazeHeight || g.isCollision(newHead) {
				g.state = StateGameOver
				return nil
			}

			g.snake = append([]Point{newHead}, g.snake...)

			if newHead == g.exit {
				g.state = StateWin
				return nil
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
	case StateGameOver, StateWin:
		if ebiten.IsKeyPressed(ebiten.KeyR) {
			g = NewGame()
			g.state = StatePlaying
		}
	}

	return nil
}

func (g *Game) handleInput() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) && g.direction.X == 0 {
		g.direction = Point{X: -1, Y: 0}
		return true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) && g.direction.X == 0 {
		g.direction = Point{X: 1, Y: 0}
		return true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) && g.direction.Y == 0 {
		g.direction = Point{X: 0, Y: -1}
		return true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) && g.direction.Y == 0 {
		g.direction = Point{X: 0, Y: 1}
		return true
	}
	if (g.direction.X == -1 && inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft)) ||
		(g.direction.X == 1 && inpututil.IsKeyJustPressed(ebiten.KeyArrowRight)) ||
		(g.direction.Y == -1 && inpututil.IsKeyJustPressed(ebiten.KeyArrowUp)) ||
		(g.direction.Y == 1 && inpututil.IsKeyJustPressed(ebiten.KeyArrowDown)) {
		return true
	}
	return false
}

func drawMaze(g *Game, screen *ebiten.Image) {
	// draw walls
	for y := 0; y < g.mazeHeight; y++ {
		for x := 0; x < VIEWPORT_WIDTH; x++ {
			if x+g.viewportX < g.mazeWidth && g.maze[y][x+g.viewportX] {
				ebitenutil.DrawRect(screen, float64(x*GRID_SIZE), float64(y*GRID_SIZE), GRID_SIZE-1, GRID_SIZE-1, color.RGBA{100, 100, 100, 255})
			}
		}
	}

	// draw food
	for _, food := range g.foods {
		if food.X >= g.viewportX && food.X < g.viewportX+VIEWPORT_WIDTH {
			ebitenutil.DrawRect(screen, float64((food.X-g.viewportX)*GRID_SIZE), float64(food.Y*GRID_SIZE), GRID_SIZE-1, GRID_SIZE-1, color.RGBA{255, 0, 0, 255})
		}
	}

	// draw exit
	if g.exit.X >= g.viewportX && g.exit.X < g.viewportX+VIEWPORT_WIDTH {
		ebitenutil.DrawRect(screen, float64((g.exit.X-g.viewportX)*GRID_SIZE), float64(g.exit.Y*GRID_SIZE), GRID_SIZE-1, GRID_SIZE-1, color.RGBA{0, 0, 255, 255})
	}
}

func drawSnake(g *Game, screen *ebiten.Image) {
	for _, p := range g.snake {
		if p.X >= g.viewportX && p.X < g.viewportX+VIEWPORT_WIDTH {
			ebitenutil.DrawRect(screen, float64((p.X-g.viewportX)*GRID_SIZE), float64(p.Y*GRID_SIZE), GRID_SIZE-1, GRID_SIZE-1, color.RGBA{0, 255, 0, 255})
		}
	}
}

func drawHUD(g *Game, screen *ebiten.Image) {
	text.Draw(screen, "Score: "+string(rune(g.score+'0')), g.font, 10, 30, color.White)
	if g.state == StateGameOver {
		text.Draw(screen, "Game Over!", g.font, SCREEN_WIDTH/2-60, SCREEN_HEIGHT/2-30, color.White)
		text.Draw(screen, "Press 'R' to restart", g.font, SCREEN_WIDTH/2-100, SCREEN_HEIGHT/2+30, color.White)
	} else if g.state == StateWin {
		text.Draw(screen, "You Win!", g.font, SCREEN_WIDTH/2-50, SCREEN_HEIGHT/2-30, color.White)
		text.Draw(screen, "Press 'R' to restart", g.font, SCREEN_WIDTH/2-100, SCREEN_HEIGHT/2+30, color.White)
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 255})

	switch g.state {
	case StateStart:
		text.Draw(screen, TITLE, g.font, SCREEN_WIDTH/2-80, SCREEN_HEIGHT/2-30, color.White)
		text.Draw(screen, "Press SPACE to start", g.font, SCREEN_WIDTH/2-100, SCREEN_HEIGHT/2+30, color.White)
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
