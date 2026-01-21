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

func (h *Hub) GetAIMove(player AIPlayer, fen string, validMoves []*chess.Move) (string, error) {
	if player == PlayerChatGPT {
		return h.getChatGPTMove(fen, validMoves)
	}
	return h.getClaudeMove(fen, validMoves)
}
