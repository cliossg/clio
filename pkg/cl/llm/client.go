package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	apiKey      string
	model       string
	temperature float64
	httpClient  *http.Client
}

func NewClient(apiKey, model string, temperature float64) *Client {
	return &Client{
		apiKey:      apiKey,
		model:       model,
		temperature: temperature,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type ProofreadRequest struct {
	Text string `json:"text"`
}

type ProofreadResponse struct {
	Result      ResultBlock  `json:"result"`
	Corrections []Correction `json:"corrections"`
	Summary     string       `json:"summary"`
	Notes       Notes        `json:"notes"`
}

type ResultBlock struct {
	Mode string `json:"mode"`
	Text string `json:"text"`
}

type Correction struct {
	Type        string `json:"type"`
	Original    string `json:"original"`
	Replacement string `json:"replacement"`
	Explanation string `json:"explanation"`
}

type Notes struct {
	LanguageDetected string `json:"language_detected"`
	Confidence       string `json:"confidence"`
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

const systemPrompt = `You are a professional editorial proofreader performing a comprehensive "Proofread" operation.

Your task is to improve text through four integrated passes:
1. GRAMMAR: Fix spelling, grammar, punctuation, and syntax errors
2. STYLE: Improve clarity, flow, and reduce verbosity
3. ECHOES: Identify words/phrases repeated in close proximity
4. OVERUSE: Flag clichés and worn expressions

Core principles:
- Be conservative. Do not rewrite unless there is a clear benefit.
- Preserve the author's voice, intent, and register.
- Make the smallest change that fixes the problem.
- Every correction must be explained specifically, not generically.

Multilingual behavior:
- You are fully proficient in multiple languages and can reliably detect which language is being used.
- English is the default working language unless the text clearly belongs to another language.
- Do NOT correct quoted text, excerpts, or citations in another language.

Output format (MANDATORY):
You MUST respond with a valid JSON object. No prose before or after.

{
  "result": {
    "mode": "replace",
    "text": "<the complete proofread text>"
  },
  "corrections": [
    {
      "type": "grammar" | "style" | "echo" | "overuse",
      "original": "<exact original fragment>",
      "replacement": "<corrected fragment or empty if just flagged>",
      "explanation": "<specific explanation for THIS correction>"
    }
  ],
  "summary": "<1-2 sentence summary of what was changed>",
  "notes": {
    "language_detected": "<detected language code>",
    "confidence": "low" | "medium" | "high"
  }
}

Rules for corrections array:
- Each correction must reference the ACTUAL text, not generic advice
- "original" must be an exact quote from the input
- "explanation" must explain why THIS specific change improves the text
- Order corrections by their appearance in the text

If no corrections are needed:
- result.mode = "none"
- result.text = "" (empty)
- corrections = []
- summary = "No corrections needed."`

func (c *Client) Proofread(ctx context.Context, text string) (*ProofreadResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("LLM API key not configured")
	}

	userPrompt := fmt.Sprintf("Operation: Proofread\n\nPerform a comprehensive editorial pass on the following text.\nApply grammar, style, echo detection, and overuse flagging.\nReturn the corrected text and a detailed list of every correction made.\n\nText to proofread:\n\"\"\"\n%s\n\"\"\"", text)

	req := openAIRequest{
		Model:       c.model,
		Temperature: c.temperature,
		Messages: []openAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if openAIResp.Error != nil {
		return nil, fmt.Errorf("LLM API error: %s", openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	content := cleanMarkdownWrapper(openAIResp.Choices[0].Message.Content)

	var result ProofreadResponse
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
	}

	return &result, nil
}

func cleanMarkdownWrapper(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}

func (c *Client) IsConfigured() bool {
	return c.apiKey != ""
}
