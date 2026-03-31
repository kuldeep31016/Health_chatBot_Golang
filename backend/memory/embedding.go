package memory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"health-assistant/backend/jobs"
)

type EmbeddingClient struct {
	APIKey     string
	HTTPClient *http.Client
}

func NewEmbeddingClient(apiKey string) *EmbeddingClient {
	return &EmbeddingClient{
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 25 * time.Second},
	}
}

func (c *EmbeddingClient) EmbedText(text string) ([]float64, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("missing GEMINI_API_KEY")
	}

	var vector []float64
	err := jobs.WithRetry(jobs.RetryConfig{MaxAttempts: 3, Delay: 2 * time.Second}, func() error {
		body := map[string]interface{}{
			"model": "models/text-embedding-004",
			"content": map[string]interface{}{
				"parts": []map[string]string{{"text": text}},
			},
		}

		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}

		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/text-embedding-004:embedContent?key=%s", c.APIKey)
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
			return fmt.Errorf("embedding api status %d: %s", resp.StatusCode, string(respBody))
		}

		var parsed struct {
			Embedding struct {
				Values []float64 `json:"values"`
			} `json:"embedding"`
		}
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			return err
		}
		if len(parsed.Embedding.Values) == 0 {
			return fmt.Errorf("empty embedding vector")
		}

		vector = parsed.Embedding.Values
		return nil
	})

	if err != nil {
		return nil, err
	}
	return vector, nil
}
