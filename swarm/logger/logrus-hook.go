package logger

import (
    "sync"

    "github.com/sirupsen/logrus"
)

type logrusHook struct {
    MaxEntries int

    entries []logrus.Entry
    mu      sync.RWMutex
}

var hook = &logrusHook{MaxEntries: 5000}

func InstallHook() {
    logrus.AddHook(hook)
}

func GetLogs() []*logrus.Entry {
    return hook.AllEntries()
}

func (t *logrusHook) Fire(e *logrus.Entry) error {
    t.mu.Lock()
    defer t.mu.Unlock()
    t.entries = append(t.entries, *e)
    if len(t.entries) > t.MaxEntries {
        t.entries = t.entries[len(t.entries)-t.MaxEntries:]
    }
    return nil
}

func (t *logrusHook) Levels() []logrus.Level {
    return logrus.AllLevels
}

// LastEntry returns the last entry that was logged or nil.
func (t *logrusHook) LastEntry() *logrus.Entry {
    t.mu.RLock()
    defer t.mu.RUnlock()
    i := len(t.entries) - 1
    if i < 0 {
        return nil
    }
    return &t.entries[i]
}

// AllEntries returns all entries that were logged.
func (t *logrusHook) AllEntries() []*logrus.Entry {
    t.mu.RLock()
    defer t.mu.RUnlock()
    // Make a copy so the returned value won't race with future log requests
    entries := make([]*logrus.Entry, len(t.entries))
    for i := 0; i < len(t.entries); i++ {
        // Make a copy, for safety
        entries[i] = &t.entries[i]
    }
    return entries
}

// Reset removes all entries from this test hook.
func (t *logrusHook) Reset() {
    t.mu.Lock()
    defer t.mu.Unlock()
    t.entries = make([]logrus.Entry, 0)
}
