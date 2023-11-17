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
	"slices"
	"sync"
	"sync/atomic"

	"github.com/amanofbits/gogoevents/internal/wildcard"
)

type Bus[EData any] struct {
	topicCache map[string][]uint32
	patterns   []string
	subs       []Subscriber[EData]
	mu         sync.RWMutex
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

	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.patterns) == 0 {
		wg.Add(0)
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

	ev := Event[EData]{Topic: topic, Data: &data, wg: &wg}
	dones := make([]atomic.Bool, len(indices))

	for i := 0; i < len(indices); i++ {
		go handle(b.subs[indices[i]], ev, &dones[i])
	}
	return &wg, nil
}

func handle[EData any](sub Subscriber[EData], ev Event[EData], done *atomic.Bool) {
	ev.done = done
	sub.handler(ev)
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
	b.patterns = slices.Insert(b.patterns, pos, pattern)

	b.subs = slices.Insert(b.subs, pos, Subscriber[EData]{
		handler: handler,
		id:      newUniqueId(),
	})
	return b.subs[pos]
}

// Removes specified subscriber from subscriptions.
func (b *Bus[EData]) Unsubscribe(sub Subscriber[EData]) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	clear(b.topicCache)

	idx := 0
	for ; idx < len(b.subs); idx++ {
		if b.subs[idx].id == sub.id {
			break
		}
	}

	b.subs = append(b.subs[:idx], b.subs[idx:]...)
	return true
}

// Modified copy from slices.BinarySearchFunc, Go 1.21.4, with element index.
// Why not include item's index into comparison function?..
//
// The slice must be sorted in increasing order, where "increasing"
// is defined by cmp. cmp should return 0 if the slice element matches
// the target, a negative number if the slice element precedes the target,
// or a positive number if the slice element follows the target.
// cmp must implement the same ordering as the slice, such that if
// cmp(a, t) < 0 and cmp(b, t) >= 0, then a must precede b in the slice.
func binarySearchFunc[S ~[]E, E, T any](x S, target T, cmp func(E, T, int) int) (int, bool) {
	n := len(x)
	// Define cmp(x[-1], target) < 0 and cmp(x[n], target) >= 0 .
	// Invariant: cmp(x[i - 1], target) < 0, cmp(x[j], target) >= 0.
	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1) // avoid overflow when computing h
		// i ≤ h < j
		if cmp(x[h], target, h) < 0 {
			i = h + 1 // preserves cmp(x[i - 1], target) < 0
		} else {
			j = h // preserves cmp(x[j], target) >= 0
		}
	}
	// i == j, cmp(x[i-1], target) < 0, and cmp(x[j], target) (= cmp(x[i], target)) >= 0  =>  answer is i.
	return i, i < n && cmp(x[i], target, i) == 0
}
