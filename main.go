package main

import (
	"bytes"
	"embed"
	"fmt"
	"image/color"
	"log"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	SCREEN_WIDTH   = 640
	SCREEN_HEIGHT  = 480
	GRID_SIZE      = 20
	VIEWPORT_WIDTH = SCREEN_WIDTH / GRID_SIZE
	TITLE          = "PACSNEK MAZE"
	POWERUP_TIME   = 300 // 5 seconds @ 60fps
)

type Slice[E any] []E

func NewSlice[E any](elements ...E) Slice[E] {
	return elements
}

func (slice Slice[E]) removeAt(index int) Slice[E] {
	return append(slice[:index], slice[index+1:]...)
}

type Status int

const (
	StatusStarted Status = iota
	StatusPlaying
	StatusLost
	StatusWon
)

//go:embed assets/*
var assets embed.FS

// global font
var font Font = NewFont()

type Font struct {
	regular text.GoTextFace
	small   text.GoTextFace
}

// NewFont creates a new Font struct by loading the font from the assets folder
func NewFont() Font {
	fontBytes, err := assets.ReadFile("assets/pressstart2p.ttf")
	if err != nil {
		log.Fatal(err)
	}

	fontFaceSource, err := text.NewGoTextFaceSource(bytes.NewReader(fontBytes))
	if err != nil {
		log.Fatal(err)
	}

	return Font{
		regular: text.GoTextFace{
			Source: fontFaceSource,
			Size:   24,
		},
		small: text.GoTextFace{
			Source: fontFaceSource,
			Size:   20,
		},
	}
}

// state is the global game state, initialized with a new State instance. it
// holds all the current game information and is updated throughout gameplay.
var state State = NewState()

// NewState creates and returns a new State instance, initializing the game with
// default values for a new game session.
func NewState() State {
	level := NewLevel(1)

	return State{
		status:       StatusStarted,
		viewportX:    0,
		level:        level,
		snake:        NewSnake(level.entrance),
		score:        0,
		powerUpTimer: 0,
	}
}

// Vec2 represents a 2D vector or point with integer coordinates. it's used
// throughout the game to represent position and direction.
type Vec2 struct {
	x int
	y int
}

type Snake struct {
	body                Slice[Vec2]
	prevDirection       Vec2
	direction           Vec2
	framesSinceLastMove int
}

func NewSnake(position Vec2) Snake {
	return Snake{
		body:                NewSlice(position),
		prevDirection:       Vec2{x: 1, y: 0},
		direction:           Vec2{x: 0, y: 0},
		framesSinceLastMove: 0,
	}
}

// createHead calculates the new position for the snake's head based on its
// current position and direction. it wraps around the level boundaries to
// create a toroidal world effect.
func (snake *Snake) createHead() Vec2 {
	head := snake.getHead()
	prev := snake.prevDirection
	height := state.level.height
	width := state.level.width
	return Vec2{
		x: (head.x + prev.x + width) % width,
		y: (head.y + prev.y + height) % height,
	}
}

func (snake *Snake) move() {
	// only move every 10 frames
	snake.framesSinceLastMove += 1
	if snake.framesSinceLastMove < 10 {
		return
	}
	snake.framesSinceLastMove = 0

	if snake.direction.x != -snake.prevDirection.x || snake.direction.y != -snake.prevDirection.y {
		snake.prevDirection = snake.direction
	}

	newHead := snake.createHead()

	snake.checkCollision(newHead)

	snake.prepend(newHead)

	if newHead == state.level.exit {
		state.status = StatusWon
		return
	}

	snake.eatFood()
}

func (snake *Snake) checkCollision(head Vec2) {
	if state.level.walls[head.y][head.x] {
		state.status = StatusLost
		return
	}
	for _, s := range snake.getTail() {
		if s == head {
			state.status = StatusLost
			return
		}
	}
}

func (snake *Snake) eatFood() {
	for i, foodPosition := range state.level.foods {
		if snake.getHead() == foodPosition {
			state.level.foods = state.level.foods.removeAt(i)
			state.score++
			state.powerUpTimer = POWERUP_TIME
			return
		}
	}
	snake.removeLastSegment()
}

func (snake *Snake) prepend(newHead Vec2) {
	snake.body = append(NewSlice(newHead), snake.body...)
}

// removeLastSegment removes the last segment of the snake's body. this function
// is called when the snake moves without eating food, effectively making the
// snake appear to move forward by removing its tail as a new head segment is
// added.
func (snake *Snake) removeLastSegment() {
	// Remove the last element of the body slice by re-slicing to exclude the
	// final element.
	snake.body = snake.body[:len(snake.body)-1]
}

func (snake *Snake) getHead() Vec2 {
	return snake.body[0]
}

func (snake *Snake) getTail() Slice[Vec2] {
	return snake.body[1:]
}

type Level struct {
	id       int
	walls    Slice[Slice[bool]]
	foods    Slice[Vec2]
	entrance Vec2
	exit     Vec2
	width    int
	height   int
}

// NewLevel creates a new instance of Level from the given id by loading the
// associated text file from the assets folder
func NewLevel(id int) Level {
	level := Level{id: id}
	filename := fmt.Sprintf("assets/level-%d.txt", id)
	content, err := assets.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	levelString := string(content)

	lines := strings.Split(strings.TrimSpace(levelString), "\n")
	level.height = len(lines)
	level.width = len(lines[0])
	level.walls = make(Slice[Slice[bool]], level.height)
	level.foods = Slice[Vec2]{}

	for y, line := range lines {
		level.walls[y] = make(Slice[bool], level.width)
		for x, char := range line {
			switch char {
			case '#':
				level.walls[y][x] = true
			case 'F':
				level.foods = append(level.foods, Vec2{x: x, y: y})
			case 'S':
				level.entrance = Vec2{x: x, y: y}
			case 'E':
				level.exit = Vec2{x: x, y: y}
			}
		}
	}

	if level.entrance == (Vec2{}) || level.exit == (Vec2{}) || len(level.foods) == 0 {
		log.Fatal("Invalid level: missing snake start, exit, or food")
	}

	return level
}

var startBlinkCounter int = 0

type State struct {
	snake        Snake
	level        Level
	status       Status
	score        int
	viewportX    int
	powerUpTimer int
}

func handleInput() {
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) && state.snake.prevDirection.x == 0 {
		state.snake.direction = Vec2{x: -1, y: 0}
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) && state.snake.prevDirection.x == 0 {
		state.snake.direction = Vec2{x: 1, y: 0}
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) && state.snake.prevDirection.y == 0 {
		state.snake.direction = Vec2{x: 0, y: -1}
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) && state.snake.prevDirection.y == 0 {
		state.snake.direction = Vec2{x: 0, y: 1}
	}
}

// updateViewport adjusts the viewport x to follow the snake when it is past a
// certain the center of the viewport by a certain amount
func updateViewport() {
	head := state.snake.getHead()
	if head.x-state.viewportX > VIEWPORT_WIDTH*3/4 {
		state.viewportX = head.x - VIEWPORT_WIDTH*3/4
	} else if head.x-state.viewportX < VIEWPORT_WIDTH/4 {
		state.viewportX = head.x - VIEWPORT_WIDTH/4
	}

	if state.viewportX < 0 {
		state.viewportX = 0
	} else if state.viewportX > state.level.width-VIEWPORT_WIDTH {
		state.viewportX = state.level.width - VIEWPORT_WIDTH
	}
}

func main() {
	ebiten.SetWindowSize(SCREEN_WIDTH, SCREEN_HEIGHT)
	ebiten.SetWindowTitle(TITLE)

	game := &Game{}
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

// Empty struct to satisfy ebitengine interface
type Game struct{}

// satisfies the main layout method from the [ebiten.Game] interface
//
// [ebiten.Game]: https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2#Game
func (*Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return SCREEN_WIDTH, SCREEN_HEIGHT
}

// satisfies the main drawing method from [ebiten.Game]
//
// [ebiten.Game]: https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2#Game
func (*Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 255})

	switch state.status {
	case StatusStarted:
		drawStartScreen(screen)
	case StatusPlaying, StatusLost, StatusWon:
		drawLevel(screen)
		drawSnake(screen)
		drawHUD(screen)
	}
}

func drawStartScreen(screen *ebiten.Image) {
	titleWidth := float64(len(TITLE)) * font.regular.Size
	titleHeight := font.regular.Size

	op := &text.DrawOptions{}
	op.GeoM.Translate((float64(SCREEN_WIDTH)-titleWidth)/2, float64(SCREEN_HEIGHT)/2-titleHeight/2-30)
	text.Draw(screen, TITLE, &font.regular, op)

	if startBlinkCounter < 30 {
		startText := "press SPACE to start"
		startWidth := float64(len(startText)) * font.regular.Size

		op.GeoM.Reset()
		op.GeoM.Translate((float64(SCREEN_WIDTH)-startWidth)/2, float64(SCREEN_HEIGHT)/2+30)
		text.Draw(screen, startText, &font.regular, op)
	}
}

func drawLevel(screen *ebiten.Image) {
	for y := 0; y < state.level.height; y++ {
		for x := 0; x < VIEWPORT_WIDTH; x++ {
			worldX := x + state.viewportX
			if worldX < state.level.width && state.level.walls[y][worldX] {
				vector.DrawFilledRect(screen, float32(x*GRID_SIZE), float32(y*GRID_SIZE), GRID_SIZE-1, GRID_SIZE-1, color.RGBA{100, 100, 100, 255}, true)
			}
		}
	}

	for _, food := range state.level.foods {
		if food.x >= state.viewportX && food.x < state.viewportX+VIEWPORT_WIDTH {
			vector.DrawFilledRect(screen, float32((food.x-state.viewportX)*GRID_SIZE), float32(food.y*GRID_SIZE), GRID_SIZE-1, GRID_SIZE-1, color.RGBA{255, 0, 0, 255}, true)
		}
	}

	if state.level.exit.x >= state.viewportX && state.level.exit.x < state.viewportX+VIEWPORT_WIDTH {
		vector.DrawFilledRect(screen, float32((state.level.exit.x-state.viewportX)*GRID_SIZE), float32(state.level.exit.y*GRID_SIZE), GRID_SIZE-1, GRID_SIZE-1, color.RGBA{0, 0, 255, 255}, true)
	}
}

func drawSnake(screen *ebiten.Image) {
	for _, p := range state.snake.body {
		if p.x >= state.viewportX && p.x < state.viewportX+VIEWPORT_WIDTH {
			vector.DrawFilledRect(screen, float32((p.x-state.viewportX)*GRID_SIZE), float32(p.y*GRID_SIZE), GRID_SIZE-1, GRID_SIZE-1, color.RGBA{0, 255, 0, 255}, true)
		}
	}
}

func drawHUD(screen *ebiten.Image) {
	// draw score
	op := &text.DrawOptions{}
	op.GeoM.Translate(10, 25)
	text.Draw(screen, "score: "+strconv.Itoa(state.score), &font.small, op)

	// draw power up timer
	if state.powerUpTimer > 0 {
		powerUpText := "power-up: " + strconv.Itoa(state.powerUpTimer/60) // Convert frames to seconds
		op.GeoM.Translate(0, 25)
		text.Draw(screen, powerUpText, &font.small, op)
	}

	// draw end game message
	if state.status == StatusLost || state.status == StatusWon {
		// semi-transparent black background
		vector.DrawFilledRect(screen, 0, 0, SCREEN_WIDTH, SCREEN_HEIGHT, color.RGBA{0, 0, 0, 128}, true)

		op := &text.DrawOptions{}

		message := "game over!"
		if state.status == StatusWon {
			message = "you win!"
		}

		messageWidth := float64(len(message)) * float64(font.small.Size)
		op.GeoM.Translate((float64(SCREEN_WIDTH)-messageWidth)/2, float64(SCREEN_HEIGHT)/2-25)
		text.Draw(screen, message, &font.small, op)

		restartText := "press R to restart"
		restartWidth := float64(len(restartText)) * float64(font.small.Size)
		op.GeoM.Reset()
		op.GeoM.Translate((float64(SCREEN_WIDTH)-restartWidth)/2, float64(SCREEN_HEIGHT)/2+25)
		text.Draw(screen, restartText, &font.small, op)
	}
}

// satisfies the main update method from the [ebiten.Game] interface
//
// [ebiten.Game]: https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2#Game
func (*Game) Update() error {
	switch state.status {
	case StatusStarted:
		updateStartState()
	case StatusPlaying:
		updatePlayingState()
	case StatusLost, StatusWon:
		updateEndState()
	}
	return nil
}

func updateStartState() {
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		state.status = StatusPlaying
	}
	startBlinkCounter = (startBlinkCounter + 1) % 60
}

func updatePlayingState() {
	handleInput()
	state.snake.move()
	updateViewport()
	if state.powerUpTimer > 0 {
		state.powerUpTimer -= 1
	}
}

func updateEndState() {
	if ebiten.IsKeyPressed(ebiten.KeyR) {
		state = NewState()
		state.status = StatusPlaying
	}
}
