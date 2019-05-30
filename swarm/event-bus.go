package swarm

import (
	"context"
	"sync"
	"time"

	"github.com/Conscience/protocol/swarm/nodeeth"
)

type EventBus struct {
	watchers        map[*Watcher]struct{}
	chIncoming      chan MaybeEvent
	chAddWatcher    chan *Watcher
	chRemoveWatcher chan *Watcher
	chDone          chan struct{}

	eth         *nodeeth.Client
	repoManager *RepoManager
}

func NewEventBus(repoManager *RepoManager, eth *nodeeth.Client) *EventBus {
	bus := &EventBus{
		watchers:        make(map[*Watcher]struct{}),
		chIncoming:      make(chan MaybeEvent, 100),
		chAddWatcher:    make(chan *Watcher),
		chRemoveWatcher: make(chan *Watcher),
		chDone:          make(chan struct{}),
		eth:             eth,
		repoManager:     repoManager,
	}

	go bus.runLoop()

	return bus
}

func (bus *EventBus) Close() {
	close(bus.chDone)
}

func (bus *EventBus) runLoop() {
	for {
		select {
		case <-bus.chDone:
			return

		case event := <-bus.chIncoming:
			ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
			wg := &sync.WaitGroup{}

			for w := range bus.watchers {
				if w.IsWatching(event.EventType) {
					wg.Add(1)
					go func(w *Watcher) {
						defer wg.Done()
						w.Notify(ctx, event)
					}(w)
				}
			}
			wg.Wait()

		case watcher := <-bus.chAddWatcher:
			bus.watchers[watcher] = struct{}{}

		case watcher := <-bus.chRemoveWatcher:
			watcher.close()
			delete(bus.watchers, watcher)
		}
	}
}

func (bus *EventBus) Watch(settings *WatcherSettings) *Watcher {
	w := NewWatcher(settings, bus.eth, bus.repoManager.RepoIDList())
	bus.chAddWatcher <- w
	return w
}

func (bus *EventBus) Unwatch(w *Watcher) {
	bus.chRemoveWatcher <- w
}

func (bus *EventBus) NotifyWatchers(event MaybeEvent) {
	bus.chIncoming <- event
}

type EventType int

const numEventTypes = 4

const (
	EventType_AddedRepo EventType = 1 << iota
	EventType_PulledRepo
	EventType_PushedRepo
	EventType_UpdatedRef
	// RemovedRepo
	// ReplicationRequested
	// BecomeReplicatorRequested
	// ClonedRepo
	// BehindRemote
	// UpdatedPermissions
)

func EventTypeBitfieldToArray(bitfield EventType) []EventType {
	eventTypes := []EventType{}
	for i := uint(0); i < numEventTypes; i++ {
		if bitfield&(1<<i) != 0 {
			eventTypes = append(eventTypes, EventType(1<<i))
		}
	}
	return eventTypes
}

type (
	MaybeEvent struct {
		EventType       EventType
		AddedRepoEvent  *AddedRepoEvent
		PulledRepoEvent *PulledRepoEvent
		PushedRepoEvent *PushedRepoEvent
		UpdatedRefEvent *nodeeth.UpdatedRefEvent
		Error           error
	}

	AddedRepoEvent struct {
		RepoID   string
		RepoRoot string
	}

	PulledRepoEvent struct {
		RepoID      string
		RepoRoot    string
		UpdatedRefs []string
	}

	PushedRepoEvent struct {
		RepoID     string
		RepoRoot   string
		BranchName string
		Commit     string
	}

	WatcherSettings struct {
		EventTypes      EventType
		UpdatedRefStart uint64
	}

	Watcher struct {
		EventTypes EventType
		chOut      chan MaybeEvent
		refWatcher *nodeeth.UpdatedRefEventWatcher
		ctx        context.Context
		cancel     func()
	}
)

func NewWatcher(settings *WatcherSettings, eth *nodeeth.Client, repoIDList []string) *Watcher {
	ctx, cancel := context.WithCancel(context.Background())

	w := &Watcher{
		EventTypes: settings.EventTypes,
		chOut:      make(chan MaybeEvent),
		refWatcher: nil,
		ctx:        ctx,
		cancel:     cancel,
	}

	if w.IsWatching(EventType_UpdatedRef) {
		w.refWatcher = nodeeth.NewUpdatedRefEventWatcher(ctx, eth, repoIDList, settings.UpdatedRefStart)
		go w.passthroughUpdatedRefEvents()
	}

	return w
}

func (w *Watcher) Events() <-chan MaybeEvent {
	return w.chOut
}

func (w *Watcher) Notify(ctx context.Context, event MaybeEvent) {
	if event.AddedRepoEvent != nil && w.refWatcher != nil {
		w.refWatcher.AddRepo(ctx, event.AddedRepoEvent.RepoID)
	}

	select {
	case w.chOut <- event:
	case <-w.ctx.Done():
		// @@TODO: return an error?
	case <-ctx.Done():
		// @@TODO: return an error?
	}
}

func (w *Watcher) IsWatching(eventType EventType) bool {
	return w.EventTypes&eventType != 0
}

func (w *Watcher) close() {
	w.cancel()
}

func (w *Watcher) passthroughUpdatedRefEvents() {
	for maybeEvt := range w.refWatcher.Events() {
		if maybeEvt.Error != nil {
			select {
			case w.chOut <- MaybeEvent{Error: maybeEvt.Error}:
			case <-w.ctx.Done():
			}
			return
		}

		select {
		case w.chOut <- MaybeEvent{EventType: EventType_UpdatedRef, UpdatedRefEvent: &maybeEvt.Event}:
		case <-w.ctx.Done():
			return
		}
	}
}
