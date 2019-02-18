package swarm

import (
	"context"

	"github.com/Conscience/protocol/swarm/nodeeth"
)

type EventType int

const (
	AddedRepo EventType = iota
	PulledRepo
	UpdatedRef
	// RemovedRepo
	// ReplicationRequested
	// BecomeReplicatorRequested
	// ClonedRepo
	// CheckpointedRepo
	// BehindRemote
	// UpdatedPermissions
)

type MaybeEvent struct {
	EventType       EventType
	AddedRepoEvent  *AddedRepoEvent
	PulledRepoEvent *PulledRepoEvent
	UpdatedRefEvent *UpdatedRefEvent
	Error           error
}

type AddedRepoEvent struct {
	RepoID   string
	RepoRoot string
}

type PulledRepoEvent struct {
	RepoID   string
	RepoRoot string
	NewHEAD  string
}

type UpdatedRefEvent struct {
	RefLog nodeeth.RefLog
}

type WatcherSettings struct {
	EventTypes      []EventType
	UpdatedRefStart uint64
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

func (w *Watcher) Notify(event MaybeEvent) {
	// if eventType == RepoAdded && w.refWatcher != nil {
	// 	w.refWatcher.AddRepo(repoID)
	// }
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
			EventType: UpdatedRef,
			UpdatedRefEvent: &UpdatedRefEvent{
				RefLog: maybeLog.Log,
			},
		}
	}
}
