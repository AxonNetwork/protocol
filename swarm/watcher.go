package swarm

import (
	"context"

	"github.com/Conscience/protocol/swarm/nodeeth"
)

type EventType int

const (
	AddedRepo EventType = iota
	PulledRepo
	PushedRepo
	UpdatedRef
	// RemovedRepo
	// ReplicationRequested
	// BecomeReplicatorRequested
	// ClonedRepo
	// BehindRemote
	// UpdatedPermissions
)

type MaybeEvent struct {
	EventType       EventType
	AddedRepoEvent  *AddedRepoEvent
	PulledRepoEvent *PulledRepoEvent
	PushedRepoEvent *PushedRepoEvent
	UpdatedRefEvent *nodeeth.UpdatedRefEvent
	Error           error
}

type AddedRepoEvent struct {
	RepoID   string
	RepoRoot string
}

type PulledRepoEvent struct {
	RepoID      string
	RepoRoot    string
	UpdatedRefs []string
}

type PushedRepoEvent struct {
	RepoID     string
	RepoRoot   string
	BranchName string
	Commit     string
}

type WatcherSettings struct {
	EventTypes      []EventType
	UpdatedRefStart uint64
}

type Watcher struct {
	EventTypes []EventType
	EventCh    chan MaybeEvent
	refWatcher *nodeeth.UpdatedRefEventWatcher
}

func NewWatcher(ctx context.Context, settings *WatcherSettings) *Watcher {
	eventCh := make(chan MaybeEvent)

	return &Watcher{
		EventTypes: settings.EventTypes,
		EventCh:    eventCh,
	}
}

func (w *Watcher) Notify(event MaybeEvent) {
	if event.AddedRepoEvent != nil && w.refWatcher != nil {
		go w.refWatcher.AddRepo(event.AddedRepoEvent.RepoID)
	}
	w.EventCh <- event
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

func (w *Watcher) AddUpdatedRefEventWatcher(rw *nodeeth.UpdatedRefEventWatcher) {
	w.refWatcher = rw
	for maybeEvt := range rw.Ch {
		if maybeEvt.Error != nil {
			w.EventCh <- MaybeEvent{Error: maybeEvt.Error}
			return
		}
		w.EventCh <- MaybeEvent{
			EventType:       UpdatedRef,
			UpdatedRefEvent: &maybeEvt.Event,
		}
	}
}
