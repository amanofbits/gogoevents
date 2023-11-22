/*
 * Holds main event bus functions
 *
 * Copyright © 2023 amanofbits
 *
 * This file is part of gogoevents.
 *
 * gogoevents is free software: you can redistribute it and/or modify
 * it under the terms of the BSD 3-Clause License as published by the
 * University of California. See the `LICENSE` file for more details.
 *
 * You should have received a copy of the BSD 3-Clause License along with
 * gogoevents. If not, see <https://opensource.org/licenses/BSD-3-Clause>.
 */

package gogoevents

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/amanofbits/gogoevents/internal/wildcard"
)

type Bus[EData any] struct {
	unhandledSink func(Event[EData])
	topicCache    map[string][]uint32
	patterns      []string
	subs          []Subscriber[EData]
	mu            sync.RWMutex
}

func NewUntyped() *Bus[any] {
	return New[any]()
}

func New[EData any]() *Bus[EData] {
	return &Bus[EData]{topicCache: make(map[string][]uint32)}
}

func (b *Bus[EData]) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	clear(b.topicCache)
	b.patterns = b.patterns[:0]
	b.subs = b.subs[:0]
	b.unhandledSink = nil
	return nil
}

func (b *Bus[EData]) TotalSubscribers() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.patterns)
}

var ErrIllegalWildcard = errors.New("wildcards not allowed in topic")

// Publishes event asynchronously and returns a WaitGroup that can be used to wait while all events are dispatched.
// Returns `ErrIllegalWildcard` if wildcard is found in the `topic`. No other errors.
func (b *Bus[EData]) Publish(topic string, data EData) (*sync.WaitGroup, error) {
	if wci := wildcard.Index(topic); wci >= 0 {
		return nil, fmt.Errorf("%s, at %d: %w", topic, wci, ErrIllegalWildcard)
	}

	wg := sync.WaitGroup{}
	ev := Event[EData]{Topic: topic, Data: &data, wg: &wg}

	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.patterns) == 0 {
		wg.Add(0)
		if b.unhandledSink != nil {
			wg.Add(1)
			go handle(b.unhandledSink, ev, &atomic.Bool{})
		}
		return &wg, nil
	}

	indices, ok := b.topicCache[topic]
	if !ok {
		matched := false
		indices = make([]uint32, 0, 4)

		if wildcard.Match(b.patterns[0], topic) {
			indices = append(indices, 0)
			matched = true
		}

		for i := 1; i < len(b.patterns); i++ {
			// A - sliightly slower
			// if b.patterns[i] != b.patterns[i-1] {
			// 	matched = wildcard.Match(b.patterns[i], topic)
			// }
			// if matched {
			// 	b.indices = append(b.indices, uint32(i))
			// }

			// B - sliightly faster
			matchValid := b.patterns[i] == b.patterns[i-1]
			if (matchValid && matched) ||
				(!matchValid && wildcard.Match(b.patterns[i], topic)) {
				indices = append(indices, uint32(i))
			}
		}
		b.topicCache[topic] = indices
	}
	wg.Add(len(indices))

	dones := make([]atomic.Bool, len(indices))

	for i := 0; i < len(indices); i++ {
		go handle(b.subs[indices[i]].handler, ev, &dones[i])
	}
	if b.unhandledSink != nil && len(indices) == 0 {
		go handle(b.unhandledSink, ev, &atomic.Bool{})
	}
	return &wg, nil
}

func handle[EData any](handler func(Event[EData]), ev Event[EData], done *atomic.Bool) {
	ev.done = done
	handler(ev)
	ev.Done()
}

func (b *Bus[EData]) Subscribe(pattern string, handler func(ev Event[EData])) Subscriber[EData] {
	pattern = wildcard.Normalize(pattern)

	b.mu.Lock()
	defer b.mu.Unlock()

	clear(b.topicCache)

	pos := -1
	pos, _ = binarySearchFunc(b.patterns, pattern, func(el, trgt string, idx int) int {
		if el > trgt {
			return 1
		}
		if pos < idx {
			pos = idx
			return -1
		}
		return 0
	})

	// Below code works because items are stored by value. Spares few ns.
	b.patterns = shift(b.patterns, pos)
	b.patterns[pos] = pattern

	b.subs = shift(b.subs, pos)
	b.subs[pos].handler = handler
	b.subs[pos].id = newUniqueId()

	return b.subs[pos]
}

// Removes specified subscriber from subscriptions.
func (b *Bus[EData]) Unsubscribe(sub Subscriber[EData]) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	idx := 0
	for ; idx < len(b.subs); idx++ {
		if b.subs[idx].id == sub.id {
			break
		}
	}
	if idx == len(b.subs) {
		return false
	}
	b.subs = append(b.subs[:idx], b.subs[idx+1:]...)
	b.patterns = append(b.patterns[:idx], b.patterns[idx+1:]...)
	clear(b.topicCache)
	return true
}

// Registers sink for events that are published but have no subscribers at the time.
// Simply set to nil to unregister.
func (b *Bus[EData]) SetUnhandledSink(sink func(ev Event[EData])) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.unhandledSink = sink
}
