package jobs

import (
	"fmt"
	"time"
)

type RetryConfig struct {
	MaxAttempts int
	Delay       time.Duration
}

func WithRetry(config RetryConfig, operation func() error) error {
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 1
	}

	var err error
	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		err = operation()
		if err == nil {
			return nil
		}

		if attempt < config.MaxAttempts {
			time.Sleep(config.Delay)
		}
	}

	return fmt.Errorf("all %d attempts failed: %w", config.MaxAttempts, err)
}
