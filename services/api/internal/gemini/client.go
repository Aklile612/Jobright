package gemini

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const DefaultModel = "gemini-2.0-flash"

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
		baseURL: "https://generativelanguage.googleapis.com/v1beta",
		http:    &http.Client{Timeout: 60 * time.Second},
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

type generateRequest struct {
	Contents         []content        `json:"contents"`
	SystemInstruction *content        `json:"systemInstruction,omitempty"`
	GenerationConfig generationConfig `json:"generationConfig"`
}

type content struct {
	Parts []part `json:"parts"`
}

type part struct {
	Text string `json:"text"`
}

type generationConfig struct {
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

type generateResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// Chat is used for ATS / cover letters / other smaller tasks.
func (c *Client) Chat(system, user string, maxTokens int) (string, error) {
	if !c.Enabled() {
		return "", fmt.Errorf("GEMINI_API_KEY is not configured")
	}
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	body, err := json.Marshal(generateRequest{
		SystemInstruction: &content{Parts: []part{{Text: system}}},
		Contents:          []content{{Parts: []part{{Text: user}}}},
		GenerationConfig: generationConfig{
			Temperature:     0.4,
			MaxOutputTokens: maxTokens,
		},
	})
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/models/%s:generateContent", c.baseURL, c.model)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var parsed generateResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("gemini decode: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("gemini: %s", parsed.Error.Message)
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("gemini HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned empty response")
	}
	text := strings.TrimSpace(parsed.Candidates[0].Content.Parts[0].Text)
	if text == "" {
		return "", fmt.Errorf("gemini returned empty response")
	}
	return text, nil
}

// GenerateCoverLetter kept for compatibility.
func (c *Client) GenerateCoverLetter(prompt string) (string, error) {
	return c.Chat(
		"Write a job application cover letter. Return ONLY the letter body — no markdown, no title.",
		prompt,
		900,
	)
}
