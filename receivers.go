/*
 * Holds event receiver collection
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
	"slices"
	"sync"

	"github.com/amanofbits/gogoevents/internal/valuecounter"
	"github.com/amanofbits/gogoevents/internal/wildcard"
)

type receiverCollection[EData any] struct {
	mu         sync.RWMutex
	all        []Receiver[EData]
	byPattern  map[string][]Receiver[EData]
	recvsCount *valuecounter.ValueCounter
	recvOps    chan recvOp[EData]
	closed     chan bool
	topicCache map[string][]Receiver[EData]
}

func newReceiverCollection[EData any]() *receiverCollection[EData] {
	r := &receiverCollection[EData]{
		all:        make([]Receiver[EData], 0, 4),
		byPattern:  make(map[string][]Receiver[EData], 4),
		recvsCount: valuecounter.New(),
		recvOps:    make(chan recvOp[EData]),
		closed:     make(chan bool),
		topicCache: make(map[string][]Receiver[EData]),
	}
	r.runOpsProcessor()
	return r
}

func (c *receiverCollection[EData]) runOpsProcessor() {
	go func(ops <-chan recvOp[EData], closed <-chan bool) {
	Loop:
		for {
			select {
			case op, ok := <-ops:
				if ok {
					op.do()
				}
			case <-closed:
				break Loop
			}
		}
	}(c.recvOps, c.closed)
}

// adds receiver to specified pattern.
// receiver may or may not already exist in collection.
func (c *receiverCollection[EData]) add(recv Receiver[EData], pattern string) (added bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, already := recv.patterns[pattern]; already {
		return false
	}
	c.byPattern[pattern] = append(c.byPattern[pattern], recv)
	recv.patterns[pattern] = dummy

	if !slices.ContainsFunc(c.all, recv.EqualTo) {
		c.all = append(c.all, recv)
	}
	return true
}

// Patterns do not use wildcard matching, they are matched as is.
func (c *receiverCollection[EData]) remove(recv Receiver[EData], patterns ...string) bool {
	matched := make([]string, 0, 1)

	c.mu.Lock()
	defer c.mu.Unlock()

	switch len(patterns) {
	case 0:
		for rp := range recv.patterns {
			matched = append(matched, rp)
		}
	case 1:
		matched = append(matched, patterns[0])
	default:
		for i := 0; i < len(patterns); i++ {
			_, ok := recv.patterns[patterns[i]]
			if !ok {
				continue
			}
			matched = append(matched, patterns[i])
			patterns = slices.Delete(patterns, i, i+1)
			i--
		}
	}

	if len(matched) == 0 {
		return false
	}

	clear(c.topicCache)

	for _, mp := range matched {
		recvsLen := len(c.byPattern[mp])
		c.byPattern[mp] = slices.DeleteFunc(c.byPattern[mp], recv.EqualTo)
		if recvsLen == len(c.byPattern[mp]) {
			panic("receiver wasn't removed, shouldn't happen")
		}
		delete(recv.patterns, mp)
	}

	if len(recv.patterns) == 0 {
		c.recvOps <- closeOp[EData](recv)
	}
	return true
}

func (c *receiverCollection[EData]) get(topic string) (r []Receiver[EData]) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if r, ok := c.topicCache[topic]; ok {
		return r
	}

	l, _ := c.recvsCount.MostCounted()
	l += 1

	r = make([]Receiver[EData], 0, l)

	for recvPattern, recvs := range c.byPattern {
		if topic == recvPattern || wildcard.Match(recvPattern, topic) {
			r = append(r, recvs...)
		}
	}
	c.recvsCount.IncFor(len(r))
	c.topicCache[topic] = r
	return r
}

func (c *receiverCollection[EData]) getPatterns(recv Receiver[EData]) []string {
	c.mu.RLock()
	defer c.mu.Unlock()

	ps := make([]string, len(recv.patterns))
	for p := range recv.patterns {
		ps = append(ps, p)
	}
	return ps
}

func (c *receiverCollection[EData]) close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	clear(c.byPattern)
	clear(c.topicCache)
	for _, recv := range c.all {
		clear(recv.patterns)
		c.recvOps <- closeOp[EData](recv)
	}
	c.closed <- true
}
