package main

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestRetryConversion(t *testing.T) {
	delays := []time.Duration{time.Millisecond, time.Millisecond}
	errLocked := errors.New("file is locked")

	tests := []struct {
		name        string
		errs        []error // error returned by each call; nil means success
		wantCalls   int
		wantRetries int
		wantErr     bool
	}{
		{"success", []error{nil}, 1, 0, false},
		{"success after retries", []error{errLocked, errLocked, nil}, 3, 2, false},
		{"fails after all retries", []error{errLocked, errLocked, errLocked}, 3, 2, true},
		{"unsupported file is not retried", []error{fmt.Errorf("%w: .txt", ErrUnsupportedFileType)}, 1, 0, true},
		{"missing sheets are not retried", []error{fmt.Errorf("%w: Sheet9", ErrSheetsNotFound)}, 1, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls, retries := 0, 0
			out, err := retryConversion(func() (string, error) {
				err := tt.errs[calls]
				calls++
				if err != nil {
					return "", err
				}
				return "out.pdf", nil
			}, delays, make(chan struct{}), func(attempt int, err error) {
				retries++
				if attempt != retries {
					t.Errorf("attempt = %d, want %d", attempt, retries)
				}
			})

			if calls != tt.wantCalls || retries != tt.wantRetries {
				t.Errorf("calls = %d, retries = %d, want %d, %d", calls, retries, tt.wantCalls, tt.wantRetries)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && out != "out.pdf" {
				t.Errorf("out = %q, want out.pdf", out)
			}
		})
	}
}

func TestRetryConversionStops(t *testing.T) {
	stop := make(chan struct{})
	close(stop)

	calls := 0
	start := time.Now()
	_, err := retryConversion(func() (string, error) {
		calls++
		return "", errors.New("file is locked")
	}, []time.Duration{time.Hour}, stop, func(int, error) {})

	if err == nil || calls != 1 {
		t.Errorf("calls = %d, err = %v, want 1 call and an error", calls, err)
	}
	if time.Since(start) > time.Second {
		t.Error("retry wait was not interrupted by stop")
	}
}
