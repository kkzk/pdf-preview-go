// Package watch detects modifications of converted files, so that the PDF
// can be regenerated automatically.
package watch

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	// regenerateDebounce is how long to wait after the last change before
	// regenerating, so that a burst of file events results in one conversion
	regenerateDebounce = 1 * time.Second
	// pollingInterval is the interval for polling file changes (backup for fsnotify)
	pollingInterval = 2 * time.Second
)

// Watcher watches the files of the last conversion and calls onChange when
// one of them has been modified. Changes are detected by fsnotify events in
// the watched directory and, as a backup, by polling. Any event (including
// Office lock files such as ~$Book.xlsx) only requests a check; onChange is
// called only if a tracked file's modification time has actually changed.
type Watcher struct {
	onChange func(file string)
	onError  func(err error)
	fs       *fsnotify.Watcher // nil if it could not be created

	// requestCh receives check requests (buffered, size 1)
	requestCh chan struct{}
	// stopCh is closed by Stop to stop background goroutines
	stopCh chan struct{}

	// mu guards the fields below, which are accessed from the fsnotify and
	// polling goroutines and from the app
	mu       sync.Mutex
	dir      string
	files    []string
	modTimes map[string]time.Time
	enabled  bool
}

// New creates a Watcher, enabled and tracking no files. onChange is called
// with the modified file from the Watcher's goroutine, which waits until it
// returns; onError is called on fsnotify errors.
func New(onChange func(file string), onError func(err error)) *Watcher {
	return &Watcher{
		onChange:  onChange,
		onError:   onError,
		requestCh: make(chan struct{}, 1),
		stopCh:    make(chan struct{}),
		modTimes:  make(map[string]time.Time),
		enabled:   true,
	}
}

// Start starts watching. If fsnotify is not available, changes are detected
// by polling only.
func (w *Watcher) Start() {
	if fs, err := fsnotify.NewWatcher(); err == nil {
		w.fs = fs
		go w.watchEvents()
	}
	go w.poll()
	go w.checkLoop()
}

// Stop stops watching. It must be called once, after Start.
func (w *Watcher) Stop() {
	close(w.stopCh)
	if w.fs != nil {
		w.fs.Close()
	}
}

// WatchDir watches dir for file events, instead of the previous directory
func (w *Watcher) WatchDir(dir string) error {
	if w.fs == nil {
		return fmt.Errorf("file watcher not initialized")
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.dir == dir {
		return nil
	}
	if w.dir != "" {
		w.fs.Remove(w.dir)
	}
	if err := w.fs.Add(dir); err != nil {
		return err
	}
	w.dir = dir
	return nil
}

// Track sets the files to check for modifications and their modification
// times at conversion (see StatModTimes)
func (w *Watcher) Track(files []string, modTimes map[string]time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.files = files
	w.modTimes = modTimes
}

// SetEnabled enables or disables change detection
func (w *Watcher) SetEnabled(enabled bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.enabled = enabled
}

// Enabled reports whether change detection is enabled
func (w *Watcher) Enabled() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.enabled
}

// StatModTimes returns the modification times of files
func StatModTimes(filePaths []string) map[string]time.Time {
	modTimes := make(map[string]time.Time)
	for _, filePath := range filePaths {
		if info, err := os.Stat(filePath); err == nil {
			modTimes[filePath] = info.ModTime()
		}
	}
	return modTimes
}

// watchEvents requests a check on file events in the watched directory
func (w *Watcher) watchEvents() {
	for {
		select {
		case event, ok := <-w.fs.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) != 0 {
				w.requestCheck()
			}
		case err, ok := <-w.fs.Errors:
			if !ok {
				return
			}
			w.onError(err)
		}
	}
}

// poll requests a check when a tracked file has been modified, catching
// missed fsnotify events. Runs until Stop.
func (w *Watcher) poll() {
	ticker := time.NewTicker(pollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			if w.findChangedFile() != "" {
				w.requestCheck()
			}
		}
	}
}

// requestCheck asks checkLoop to check for changes.
// Requests made while one is already pending are coalesced.
func (w *Watcher) requestCheck() {
	select {
	case w.requestCh <- struct{}{}:
	default: // a request is already pending
	}
}

// checkLoop checks for changes on request, waiting until changes have
// settled for regenerateDebounce, and calls onChange. Runs until Stop.
func (w *Watcher) checkLoop() {
	for {
		select {
		case <-w.stopCh:
			return
		case <-w.requestCh:
		}

		// Debounce: wait until no new request arrives for regenerateDebounce
		timer := time.NewTimer(regenerateDebounce)
	debounce:
		for {
			select {
			case <-w.stopCh:
				timer.Stop()
				return
			case <-w.requestCh:
				timer.Reset(regenerateDebounce)
			case <-timer.C:
				break debounce
			}
		}

		if changedFile := w.findChangedFile(); changedFile != "" {
			w.onChange(changedFile)
		}
	}
}

// findChangedFile returns the first tracked file whose modification time
// differs from the one recorded at the last conversion, or "" if none has
// changed or change detection is disabled.
func (w *Watcher) findChangedFile() string {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.enabled {
		return ""
	}
	for _, filePath := range w.files {
		info, err := os.Stat(filePath)
		if err != nil {
			continue // e.g. temporarily missing while Office saves
		}
		if lastModTime, exists := w.modTimes[filePath]; exists && !info.ModTime().Equal(lastModTime) {
			return filePath
		}
	}
	return ""
}
