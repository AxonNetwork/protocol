package nodehttp

import (
	"sync"

	"github.com/sirupsen/logrus"
)

type logrusHook struct {
	MaxEntries int

	entries []logEntry
	mu      *sync.RWMutex
}

type logEntry struct {
	Level   string
	Message string
}

var hook = &logrusHook{MaxEntries: 5000, mu: &sync.RWMutex{}}

func InstallLogrusHook() {
	logrus.AddHook(hook)
}

func getLogrusLogs() []logEntry {
	return hook.AllEntries()
}

func (t *logrusHook) Fire(e *logrus.Entry) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	var entry logEntry
	entry.Level = e.Level.String()
	entry.Message = e.Message

	t.entries = append(t.entries, entry)
	if len(t.entries) > t.MaxEntries {
		t.entries = t.entries[len(t.entries)-t.MaxEntries:]
	}
	return nil
}

func (t *logrusHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// AllEntries returns all entries that were logged.
func (t *logrusHook) AllEntries() []logEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Make a copy so the returned value won't race with future log requests
	entries := make([]logEntry, len(t.entries))
	for i := 0; i < len(t.entries); i++ {
		// Make a copy, for safety
		entries[i] = t.entries[i]
	}
	return entries
}
