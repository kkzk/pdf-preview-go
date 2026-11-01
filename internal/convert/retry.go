package convert

import (
	"errors"
	"time"
)

// Retry calls convert, retrying after each of delays while it fails with a
// retryable error. A file that has just been saved may be briefly locked or
// missing, e.g. while Office saves it or a sync client (Box Drive, OneDrive)
// uploads it. onRetry is called before each retry with the attempt number
// (1-based) and the error. Waiting stops when stop is closed.
func Retry(convert func() (string, error), delays []time.Duration, stop <-chan struct{}, onRetry func(attempt int, err error)) (string, error) {
	outputPath, err := convert()
	for i, delay := range delays {
		if err == nil || !isRetryable(err) {
			break
		}
		onRetry(i+1, err)
		select {
		case <-stop:
			return outputPath, err
		case <-time.After(delay):
		}
		outputPath, err = convert()
	}
	return outputPath, err
}

// isRetryable reports whether a conversion error may go away by retrying
// (e.g. the file is locked), as opposed to a permanent one
func isRetryable(err error) bool {
	return !errors.Is(err, ErrUnsupportedFileType) && !errors.Is(err, ErrSheetsNotFound)
}
