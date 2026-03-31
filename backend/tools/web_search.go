package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"health-assistant/backend/jobs"
)

func WebSearch(query string) (map[string]interface{}, error) {
	apiKey := os.Getenv("SERPER_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("SERPER_API_KEY not configured")
	}

	client := &http.Client{Timeout: 12 * time.Second}
	var out map[string]interface{}

	err := jobs.WithRetry(jobs.RetryConfig{MaxAttempts: 3, Delay: 2 * time.Second}, func() error {
		endpoint := "https://google.serper.dev/search?q=" + url.QueryEscape(query)
		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-API-KEY", apiKey)

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		bytes, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 300 {
			return fmt.Errorf("web search status %d: %s", resp.StatusCode, string(bytes))
		}

		if err := json.Unmarshal(bytes, &out); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return out, nil
}
