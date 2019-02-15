package swarm

import (
	"context"

	"github.com/Conscience/protocol/swarm/nodeeth"
)

type EventType int

const (
	RepoAdded EventType = iota
	PulledRepo
	RefUpdated
	ReplicationRequested
	BecomeReplicatorRequested
	// ClonedRepo
	// CheckpointedRepo
	// BehindRemote
	// UpdatedPermissions
)

type MaybeEvent struct {
	UpdatedRefEvent *UpdatedRefEvent
	Error           error
}

type UpdatedRefEvent struct {
	RefLog nodeeth.RefLog
}

type WatcherSettings struct {
	EventTypes      []EventType
	RefUpdatedStart uint64
}

type Watcher struct {
	EventTypes []EventType
	EventCh    chan MaybeEvent
	refWatcher *nodeeth.RefLogWatcher
}

func NewWatcher(ctx context.Context, settings *WatcherSettings) *Watcher {
	eventCh := make(chan MaybeEvent)

	return &Watcher{
		EventTypes: settings.EventTypes,
		EventCh:    eventCh,
	}
}

func (w *Watcher) Notify(repoID string, eventType EventType) {
	if eventType == RepoAdded && w.refWatcher != nil {
		w.refWatcher.AddRepo(repoID)
	}
	// w.EventCh <- MaybeEvent{RepoID: repoID, EventType: eventType}
}

func (w *Watcher) IsWatching(eventType EventType) bool {
	for _, t := range w.EventTypes {
		if t == eventType {
			return true
		}
	}
	return false
}

func (w *Watcher) Close() {
	close(w.EventCh)
}

func (w *Watcher) AddRefLogWatcher(rw *nodeeth.RefLogWatcher) {
	w.refWatcher = rw
	for maybeLog := range rw.Ch {
		if maybeLog.Error != nil {
			w.EventCh <- MaybeEvent{Error: maybeLog.Error}
			return
		}
		w.EventCh <- MaybeEvent{
			UpdatedRefEvent: &UpdatedRefEvent{
				RefLog: maybeLog.Log,
			},
		}
	}
}
