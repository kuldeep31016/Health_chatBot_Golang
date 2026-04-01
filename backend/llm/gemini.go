package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"health-assistant/backend/jobs"
)

type GeminiClient struct {
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

func NewGeminiClient(apiKey, model string) *GeminiClient {
	if model == "" {
		model = "gemini-2.5-flash"
	}
	return &GeminiClient{
		APIKey:     apiKey,
		Model:      model,
		HTTPClient: &http.Client{Timeout: 35 * time.Second},
	}
}

func (c *GeminiClient) GenerateResponse(profile, healthData, memoryContext map[string]interface{}, query string) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("missing GEMINI_API_KEY")
	}

	systemPrompt := "You are a personal AI health assistant. Use ONLY the provided user data and memory context to answer. Do NOT hallucinate. If data is not available, say so clearly. Use plain simple text only. Do NOT use markdown symbols like *, **, #, -, or backticks. Do NOT use bullet points. Keep the response clean, readable, and professional."

	payloadContext := map[string]interface{}{
		"user_profile":    profile,
		"health_data":     healthData,
		"relevant_memory": memoryContext,
		"user_query":      query,
	}

	ctxBytes, _ := json.MarshalIndent(payloadContext, "", "  ")
	fullPrompt := fmt.Sprintf("%s\n\nContext:\n%s", systemPrompt, string(ctxBytes))

	var out string
	err := jobs.WithRetry(jobs.RetryConfig{MaxAttempts: 3, Delay: 2 * time.Second}, func() error {
		body := map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"role": "user",
					"parts": []map[string]string{{"text": fullPrompt}},
				},
			},
		}

		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}

		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.Model, c.APIKey)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 300 {
			return fmt.Errorf("gemini api status %d: %s", resp.StatusCode, string(respBody))
		}

		var parsed struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			return err
		}
		if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
			return fmt.Errorf("empty llm response")
		}
		out = parsed.Candidates[0].Content.Parts[0].Text
		return nil
	})

	if err != nil {
		return "", err
	}
	return out, nil
}
