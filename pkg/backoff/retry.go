package backoff

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

// RetryWithBackoff retries a function with exponential backoff
func RetryWithBackoff(operation func() error) error {
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 10 * time.Second
	return backoff.Retry(operation, bo)
}
