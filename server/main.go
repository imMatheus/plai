package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
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
			client.send <- Message{
				Type:          "game_state",
				FEN:           h.currentGame.FEN(),
				Turn:          turn,
				SVG:           svg,
				ViewerCount:   viewerCount,
				WhitePlayer:   string(h.whitePlayer),
				BlackPlayer:   string(h.blackPlayer),
				CurrentPlayer: currentPlayer,
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

func (h *Hub) getChatGPTMove(fen string, validMoves []*chess.Move) (string, error) {
	if h.aiConfig.openAIKey == "" {
		return "", fmt.Errorf("OpenAI API key not configured")
	}

	// Convert valid moves to SAN notation
	encoder := chess.AlgebraicNotation{}
	var moveList []string
	for _, move := range validMoves {
		san := encoder.Encode(h.currentGame.Position(), move)
		moveList = append(moveList, san)
	}
	movesStr := strings.Join(moveList, ", ")

	prompt := fmt.Sprintf(`You are playing chess. The current position is: %s

Here are ALL the valid moves you can make: %s

Please respond with ONLY ONE move from the list above. Pick the move you think is best.
Do not include any explanation, just the move.`, fen, movesStr)

	reqBody := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.aiConfig.openAIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API error: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	choices := result["choices"].([]interface{})
	if len(choices) == 0 {
		return "", fmt.Errorf("no response from ChatGPT")
	}

	message := choices[0].(map[string]interface{})["message"].(map[string]interface{})
	move := strings.TrimSpace(message["content"].(string))

	return move, nil
}

func (h *Hub) getClaudeMove(fen string, validMoves []*chess.Move) (string, error) {
	if h.aiConfig.anthropicKey == "" {
		return "", fmt.Errorf("Anthropic API key not configured")
	}

	// Convert valid moves to SAN notation
	encoder := chess.AlgebraicNotation{}
	var moveList []string
	for _, move := range validMoves {
		san := encoder.Encode(h.currentGame.Position(), move)
		moveList = append(moveList, san)
	}
	movesStr := strings.Join(moveList, ", ")

	prompt := fmt.Sprintf(`You are playing chess. The current position is: %s

Here are ALL the valid moves you can make: %s

Please respond with ONLY ONE move from the list above. Pick the move you think is best.
Do not include any explanation, just the move.`, fen, movesStr)

	reqBody := map[string]interface{}{
		"model":      "claude-sonnet-4-5",
		"max_tokens": 100,
		"messages": []map[string]interface{}{
			{"role": "user", "content": prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", h.aiConfig.anthropicKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Anthropic API error: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	content := result["content"].([]interface{})
	if len(content) == 0 {
		return "", fmt.Errorf("no response from Claude")
	}

	move := strings.TrimSpace(content[0].(map[string]interface{})["text"].(string))

	return move, nil
}

func (h *Hub) getAIMove(player AIPlayer, fen string, validMoves []*chess.Move) (string, error) {
	if player == PlayerChatGPT {
		return h.getChatGPTMove(fen, validMoves)
	}
	return h.getClaudeMove(fen, validMoves)
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
	h.mu.Lock()
	defer h.mu.Unlock()

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
		return
	}

	// Get legal moves
	validMoves := h.currentGame.ValidMoves()
	if len(validMoves) == 0 {
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
	log.Printf("%s's turn to move (FEN: %s)", currentPlayer, fen)

	var move *chess.Move
	var moveErr error

	// Try up to 3 times to get a valid move from AI
	for attempt := 1; attempt <= 3; attempt++ {
		moveStr, err := h.getAIMove(currentPlayer, fen, validMoves)
		if err != nil {
			log.Printf("Attempt %d: Error getting move from %s: %v", attempt, currentPlayer, err)
			moveErr = err
			continue
		}

		log.Printf("Attempt %d: %s suggested move: %s", attempt, currentPlayer, moveStr)

		move, moveErr = h.validateAndParseMove(moveStr)
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

	// Make the move
	err := h.currentGame.Move(move)
	if err != nil {
		log.Printf("Error making move: %v", err)
		return
	}

	turn := "white"
	nextPlayer := string(h.whitePlayer)
	if h.currentGame.Position().Turn() == chess.Black {
		turn = "black"
		nextPlayer = string(h.blackPlayer)
	}

	svg := h.generateSVG()

	// Broadcast move to all clients
	h.broadcast <- Message{
		Type:          "move",
		FEN:           h.currentGame.FEN(),
		Move:          move.String(),
		Turn:          turn,
		SVG:           svg,
		ViewerCount:   len(h.clients),
		WhitePlayer:   string(h.whitePlayer),
		BlackPlayer:   string(h.blackPlayer),
		CurrentPlayer: nextPlayer,
	}

	log.Printf("Move played: %s, New FEN: %s, Next turn: %s (%s)", move.String(), h.currentGame.FEN(), turn, nextPlayer)
}

const MOVE_INTERVAL = 4_000 * time.Millisecond // 0.4 seconds

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
