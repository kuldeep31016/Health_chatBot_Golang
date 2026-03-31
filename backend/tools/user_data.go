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

	selected := map[string]interface{}{}
	keys := []string{
		"name",
		"age",
		"height_cm",
		"weight_kg",
		"hair_color",
		"eye_color",
		"gender",
		"blood_group",
		"goals",
		"allergies",
		"diet_preferences",
	}
	for _, k := range keys {
		if v, ok := userProfile[k]; ok {
			selected[k] = v
		}
	}

	return selected, nil
}

func RawProfile() map[string]interface{} {
	profileMu.RLock()
	defer profileMu.RUnlock()
	return userProfile
}
