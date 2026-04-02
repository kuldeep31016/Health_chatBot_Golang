package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"health-assistant/backend/jobs"
)

var (
	profileMu   sync.RWMutex
	userProfile map[string]interface{}
)

func LoadUserProfile(path string) error {
	var loaded map[string]interface{}

	err := jobs.WithRetry(jobs.RetryConfig{MaxAttempts: 3, Delay: 2 * time.Second}, func() error {
		bytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if err := json.Unmarshal(bytes, &loaded); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to load user profile: %w", err)
	}

	profileMu.Lock()
	defer profileMu.Unlock()
	userProfile = loaded
	return nil
}

func GetUserProfile() (map[string]interface{}, error) {
	profileMu.RLock()
	defer profileMu.RUnlock()

	if userProfile == nil {
		return nil, fmt.Errorf("user profile not loaded")
	}

	full := make(map[string]interface{}, len(userProfile))
	for k, v := range userProfile {
		full[k] = v
	}

	return full, nil
}

func RawProfile() map[string]interface{} {
	profileMu.RLock()
	defer profileMu.RUnlock()
	return userProfile
}
