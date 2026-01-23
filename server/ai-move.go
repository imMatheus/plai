package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/notnil/chess"
)

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

	// System prompt to establish chess expertise
	systemPrompt := `You are a strong chess player with deep understanding of chess strategy and tactics. 
You analyze positions carefully, considering:
- Material balance and piece activity
- King safety and pawn structure  
- Tactical opportunities (forks, pins, skewers, discovered attacks)
- Positional advantages (control of center, open files, weak squares)
- Short-term tactics vs long-term strategy

Always choose the move that gives you the best advantage.`

	userPrompt := fmt.Sprintf(`Current position (FEN): %s

Legal moves: %s

Analyze this position and choose your best move. Consider:
1. Are there any tactical opportunities (checks, captures, threats)?
2. What is the opponent threatening?
3. How can you improve your position?

Respond with ONLY the move in standard algebraic notation, nothing else.`, fen, movesStr)

	reqBody := map[string]interface{}{
		"model": "gpt-4o", // Use GPT-4o instead of mini for better chess understanding
		"messages": []map[string]interface{}{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.3, // Lower temperature for more focused/deterministic moves
		"max_tokens":  50,  // We only need a short response
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

	client := &http.Client{Timeout: 45 * time.Second} // Longer timeout for GPT-4
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

	systemPrompt := `You are a strong chess player. You MUST respond with ONLY a single chess move in algebraic notation.
	DO NOT include any explanation, analysis, or additional text.
	DO NOT say things like "I need to analyze" or "Let me think".
	Your entire response must be exactly one move from the legal moves provided, nothing more.`

	userPrompt := fmt.Sprintf(`Position (FEN): %s

Legal moves: %s

Choose your best move considering tactics, threats, and positional advantages.

CRITICAL: Respond with ONLY the move itself (e.g., "e4" or "Nf3"). No other text whatsoever.`, fen, movesStr)

	reqBody := map[string]interface{}{
		"model":       "claude-sonnet-4-20250514",
		"max_tokens":  10, // Drastically reduce to discourage explanations
		"temperature": 0.3,
		"system":      systemPrompt,
		"messages": []map[string]interface{}{
			{"role": "user", "content": userPrompt},
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

	client := &http.Client{Timeout: 45 * time.Second}
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

	moveText := strings.TrimSpace(content[0].(map[string]interface{})["text"].(string))

	// Extract just the move if Claude included extra text
	// Look for the first valid chess move pattern
	words := strings.Fields(moveText)
	if len(words) > 0 {
		// Return the first word, which should be the move
		return words[0], nil
	}

	return moveText, nil
}

func (h *Hub) GetAIMove(player AIPlayer, fen string, validMoves []*chess.Move) (string, error) {
	if player == PlayerChatGPT {
		return h.getChatGPTMove(fen, validMoves)
	}
	return h.getClaudeMove(fen, validMoves)
}
