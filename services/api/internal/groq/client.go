package groq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultModel favors free-tier volume: ~14.4k req/day and ~500k tokens/day.
const DefaultModel = "llama-3.1-8b-instant"

type Client struct {
	apiKey  string
	model   string
	baseURL string
	http    *http.Client
}

func NewClient(apiKey, model string) *Client {
	if model == "" {
		model = DefaultModel
	}
	return &Client{
		apiKey:  strings.TrimSpace(apiKey),
		model:   model,
		baseURL: "https://api.groq.com/openai/v1",
		http:    &http.Client{Timeout: 90 * time.Second},
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.apiKey != ""
}

func (c *Client) Model() string {
	if c == nil {
		return ""
	}
	return c.model
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) Chat(system, user string, maxTokens int) (string, error) {
	if !c.Enabled() {
		return "", fmt.Errorf("GROQ_API_KEY is not configured")
	}
	if maxTokens <= 0 {
		maxTokens = 1800
	}
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		text, err := c.chatOnce(system, user, maxTokens)
		if err == nil {
			return text, nil
		}
		lastErr = err
		wait, ok := rateLimitWait(err.Error())
		if !ok {
			return "", err
		}
		if wait > 45*time.Second {
			wait = 45 * time.Second
		}
		time.Sleep(wait + 500*time.Millisecond)
	}
	return "", lastErr
}

func (c *Client) chatOnce(system, user string, maxTokens int) (string, error) {
	body, err := json.Marshal(chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature: 0.4,
		MaxTokens:   maxTokens,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))

	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("groq decode: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("groq: %s", parsed.Error.Message)
	}
	if resp.StatusCode == 429 {
		return "", fmt.Errorf("groq: %s", strings.TrimSpace(string(raw)))
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("groq HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("groq returned empty response")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

func rateLimitWait(msg string) (time.Duration, bool) {
	low := strings.ToLower(msg)
	if !strings.Contains(low, "rate limit") && !strings.Contains(low, "tokens per minute") {
		return 0, false
	}
	// "Please try again in 19.73s"
	idx := strings.Index(low, "try again in ")
	if idx < 0 {
		return 20 * time.Second, true
	}
	rest := msg[idx+len("try again in "):]
	var secs float64
	n, _ := fmt.Sscanf(rest, "%f", &secs)
	if n != 1 || secs <= 0 {
		return 20 * time.Second, true
	}
	return time.Duration(secs * float64(time.Second)), true
}
