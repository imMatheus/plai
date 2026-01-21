package main

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/notnil/chess"
	"github.com/notnil/chess/image"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

type AIPlayer string

const (
	PlayerChatGPT AIPlayer = "ChatGPT"
	PlayerClaude  AIPlayer = "Claude"
)

type AIConfig struct {
	openAIKey    string
	anthropicKey string
}

type Hub struct {
	clients     map[*Client]bool
	broadcast   chan Message
	register    chan *Client
	unregister  chan *Client
	currentGame *chess.Game
	whitePlayer AIPlayer
	blackPlayer AIPlayer
	aiConfig    AIConfig
	mu          sync.RWMutex
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan Message
}

type Message struct {
	Type          string `json:"type"`
	FEN           string `json:"fen"`
	Move          string `json:"move,omitempty"`
	Turn          string `json:"turn"`
	SVG           string `json:"svg"`
	ViewerCount   int    `json:"viewerCount"`
	WhitePlayer   string `json:"whitePlayer"`
	BlackPlayer   string `json:"blackPlayer"`
	CurrentPlayer string `json:"currentPlayer,omitempty"`
}

func newHub() *Hub {
	// Randomly assign who goes first
	var whitePlayer, blackPlayer AIPlayer
	if rand.Intn(2) == 0 {
		whitePlayer = PlayerChatGPT
		blackPlayer = PlayerClaude
	} else {
		whitePlayer = PlayerClaude
		blackPlayer = PlayerChatGPT
	}

	log.Printf("Starting new game: White=%s, Black=%s", whitePlayer, blackPlayer)

	return &Hub{
		clients:     make(map[*Client]bool),
		broadcast:   make(chan Message),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		currentGame: chess.NewGame(),
		whitePlayer: whitePlayer,
		blackPlayer: blackPlayer,
		aiConfig: AIConfig{
			openAIKey:    os.Getenv("OPENAI_API_KEY"),
			anthropicKey: os.Getenv("ANTHROPIC_API_KEY"),
		},
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			// Send current game state to new client
			h.mu.RLock()
			turn := "white"
			currentPlayer := string(h.whitePlayer)
			if h.currentGame.Position().Turn() == chess.Black {
				turn = "black"
				currentPlayer = string(h.blackPlayer)
			}
			svg := h.generateSVG()
			viewerCount := len(h.clients)
			msg := Message{
				Type:          "game_state",
				FEN:           h.currentGame.FEN(),
				Turn:          turn,
				SVG:           svg,
				ViewerCount:   viewerCount,
				WhitePlayer:   string(h.whitePlayer),
				BlackPlayer:   string(h.blackPlayer),
				CurrentPlayer: currentPlayer,
			}
			log.Printf("Sending initial game state to new client: FEN=%s, White=%s, Black=%s, SVG length=%d",
				msg.FEN, msg.WhitePlayer, msg.BlackPlayer, len(msg.SVG))
			client.send <- msg
			h.mu.RUnlock()
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (h *Hub) generateSVG() string {
	buf := new(bytes.Buffer)

	white := color.RGBA{255, 255, 255, 255}
	gray := color.RGBA{217, 119, 87, 255} // claude color
	sqrs := image.SquareColors(white, gray)

	if err := image.SVG(buf, h.currentGame.Position().Board(), sqrs); err != nil {
		log.Printf("Error generating SVG: %v", err)
		return ""
	}
	return buf.String()
}

func (h *Hub) validateAndParseMove(moveStr string) (*chess.Move, error) {
	// Clean up the move string
	moveStr = strings.TrimSpace(moveStr)
	moveStr = strings.Trim(moveStr, "\"'")

	// Get all valid moves
	validMoves := h.currentGame.ValidMoves()

	// Create an AlgebraicNotation encoder to get SAN representation
	encoder := chess.AlgebraicNotation{}

	// Try to find a matching valid move using SAN notation
	for _, validMove := range validMoves {
		// Get the SAN representation of the move
		san := encoder.Encode(h.currentGame.Position(), validMove)

		// Compare with the AI's suggested move
		if san == moveStr || validMove.String() == moveStr {
			return validMove, nil
		}
	}

	return nil, fmt.Errorf("invalid move: %s", moveStr)
}

func (h *Hub) playMove() {
	// Lock briefly to read game state
	h.mu.Lock()

	// Check if game is over
	if h.currentGame.Outcome() != chess.NoOutcome {
		log.Println("Game over:", h.currentGame.Outcome())
		// Start new game and randomly reassign players
		h.currentGame = chess.NewGame()
		if rand.Intn(2) == 0 {
			h.whitePlayer = PlayerChatGPT
			h.blackPlayer = PlayerClaude
		} else {
			h.whitePlayer = PlayerClaude
			h.blackPlayer = PlayerChatGPT
		}
		log.Printf("Starting new game: White=%s, Black=%s", h.whitePlayer, h.blackPlayer)
		h.mu.Unlock()
		return
	}

	// Get legal moves
	validMoves := h.currentGame.ValidMoves()
	if len(validMoves) == 0 {
		h.mu.Unlock()
		return
	}

	// Determine which AI's turn it is
	var currentPlayer AIPlayer
	if h.currentGame.Position().Turn() == chess.White {
		currentPlayer = h.whitePlayer
	} else {
		currentPlayer = h.blackPlayer
	}

	fen := h.currentGame.FEN()

	// Unlock before making API calls!
	h.mu.Unlock()

	log.Printf("%s's turn to move (FEN: %s)", currentPlayer, fen)

	var move *chess.Move
	var moveErr error

	// Try up to 3 times to get a valid move from AI (no lock held during API calls)
	for attempt := 1; attempt <= 3; attempt++ {
		moveStr, err := h.GetAIMove(currentPlayer, fen, validMoves)
		if err != nil {
			log.Printf("Attempt %d: Error getting move from %s: %v", attempt, currentPlayer, err)
			moveErr = err
			continue
		}

		log.Printf("Attempt %d: %s suggested move: %s", attempt, currentPlayer, moveStr)

		// Lock briefly to validate move
		h.mu.Lock()
		move, moveErr = h.validateAndParseMove(moveStr)
		h.mu.Unlock()

		if moveErr == nil {
			break
		}

		log.Printf("Attempt %d: Invalid move from %s: %v", attempt, currentPlayer, moveErr)
	}

	// If all attempts failed, pick a random valid move
	if move == nil {
		log.Printf("All 3 attempts failed for %s, picking random move", currentPlayer)
		move = validMoves[rand.Intn(len(validMoves))]
	}

	// Lock to make the move
	h.mu.Lock()
	err := h.currentGame.Move(move)
	if err != nil {
		log.Printf("Error making move: %v", err)
		h.mu.Unlock()
		return
	}

	turn := "white"
	nextPlayer := string(h.whitePlayer)
	if h.currentGame.Position().Turn() == chess.Black {
		turn = "black"
		nextPlayer = string(h.blackPlayer)
	}

	svg := h.generateSVG()
	currentFEN := h.currentGame.FEN()
	moveStr := move.String()
	viewerCount := len(h.clients)
	whitePlayer := string(h.whitePlayer)
	blackPlayer := string(h.blackPlayer)

	// Unlock before broadcasting
	h.mu.Unlock()

	// Broadcast move to all clients
	h.broadcast <- Message{
		Type:          "move",
		FEN:           currentFEN,
		Move:          moveStr,
		Turn:          turn,
		SVG:           svg,
		ViewerCount:   viewerCount,
		WhitePlayer:   whitePlayer,
		BlackPlayer:   blackPlayer,
		CurrentPlayer: nextPlayer,
	}

	log.Printf("Move played: %s, New FEN: %s, Next turn: %s (%s)", moveStr, currentFEN, turn, nextPlayer)
}

const MOVE_INTERVAL = 2_000 * time.Millisecond // 2 second

func (h *Hub) gameLoop() {
	ticker := time.NewTicker(MOVE_INTERVAL)
	defer ticker.Stop()

	for range ticker.C {
		h.playMove()
	}
}

func (c *Client) readPump() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("readPump panic recovered: %v", r)
		}
		c.hub.unregister <- c
		c.conn.Close()
		log.Println("readPump exiting")
	}()

	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("readPump error: %v", err)
			break
		}
		// Client messages are ignored for now since game plays automatically
	}
}

func (c *Client) writePump() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("writePump panic recovered: %v", r)
		}
		c.conn.Close()
	}()

	for message := range c.send {
		log.Printf("Sending message to client: type=%s, FEN=%s", message.Type, message.FEN)
		err := c.conn.WriteJSON(message)
		if err != nil {
			log.Printf("Error sending message to client: %v", err)
			break
		}
		log.Printf("Message sent successfully: type=%s", message.Type)
	}
	log.Println("writePump exiting")
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	log.Printf("WebSocket connection attempt from %s", r.RemoteAddr)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	log.Printf("WebSocket connection established from %s", r.RemoteAddr)

	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan Message, 256),
	}

	// Start goroutines first
	go client.writePump()
	go client.readPump()

	// Then register the client (which triggers sending initial state)
	client.hub.register <- client

	log.Printf("Client registered from %s", r.RemoteAddr)
}

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	hub := newHub()
	go hub.run()
	go hub.gameLoop() // Start the automatic game loop

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
