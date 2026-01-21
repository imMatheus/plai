package main

import (
	"bytes"
	"image/color"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/notnil/chess"
	"github.com/notnil/chess/image"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

type Hub struct {
	clients     map[*Client]bool
	broadcast   chan Message
	register    chan *Client
	unregister  chan *Client
	currentGame *chess.Game
	mu          sync.RWMutex
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan Message
}

type Message struct {
	Type        string `json:"type"`
	FEN         string `json:"fen"`
	Move        string `json:"move,omitempty"`
	Turn        string `json:"turn"`
	SVG         string `json:"svg"`
	ViewerCount int    `json:"viewerCount"`
}

func newHub() *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		broadcast:   make(chan Message),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		currentGame: chess.NewGame(),
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
			if h.currentGame.Position().Turn() == chess.Black {
				turn = "black"
			}
			svg := h.generateSVG()
			viewerCount := len(h.clients)
			client.send <- Message{
				Type:        "game_state",
				FEN:         h.currentGame.FEN(),
				Turn:        turn,
				SVG:         svg,
				ViewerCount: viewerCount,
			}
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

func (h *Hub) playMove() {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check if game is over
	if h.currentGame.Outcome() != chess.NoOutcome {
		log.Println("Game over:", h.currentGame.Outcome())
		// Start new game
		h.currentGame = chess.NewGame()
		log.Println("Starting new game")
	}

	// Get legal moves
	moves := h.currentGame.ValidMoves()
	if len(moves) == 0 {
		return
	}

	// Play first legal move
	move := moves[rand.Intn(len(moves))]
	err := h.currentGame.Move(move)
	if err != nil {
		log.Printf("Error making move: %v", err)
		return
	}

	turn := "white"
	if h.currentGame.Position().Turn() == chess.Black {
		turn = "black"
	}

	svg := h.generateSVG()

	// Broadcast move to all clients
	h.broadcast <- Message{
		Type:        "move",
		FEN:         h.currentGame.FEN(),
		Move:        move.String(),
		Turn:        turn,
		SVG:         svg,
		ViewerCount: len(h.clients),
	}

	log.Printf("Move played: %s, New FEN: %s, Next turn: %s", move.String(), h.currentGame.FEN(), turn)
}

const MOVE_INTERVAL = 5_000 * time.Millisecond // 2 seconds

func (h *Hub) gameLoop() {
	ticker := time.NewTicker(MOVE_INTERVAL)
	defer ticker.Stop()

	for range ticker.C {
		h.playMove()
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			break
		}
		// Client messages are ignored for now since game plays automatically
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()

	for message := range c.send {
		err := c.conn.WriteJSON(message)
		if err != nil {
			break
		}
	}
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan Message, 256),
	}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}

func main() {
	hub := newHub()
	go hub.run()
	go hub.gameLoop() // Start the automatic game loop

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
