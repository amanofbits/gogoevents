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
	"log"
	"math"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/amanofbits/gogoevents/wildcard"
)

type Bus[EData any] struct {
	mu         sync.RWMutex
	recvs      map[string][]receiver[EData]
	totalRecvs atomic.Int32
}

func NewUntyped() *Bus[any] {
	return New[any]()
}

func New[EData any]() *Bus[EData] {
	return &Bus[EData]{recvs: make(map[string][]receiver[EData])}
}

func (b *Bus[EData]) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	r := make([]receiver[EData], 0, b.totalRecvs.Load())
	for _, recvs := range b.recvs {
		r = append(r, recvs...)
	}

	for _, recv := range r {
		for p := range recv.patterns {
			b.removeReceiver(recv, p)
		}
	}
	return nil
}

func (b *Bus[EData]) Subscribe(pattern string) (recv receiver[EData]) {
	b.mu.Lock()
	defer b.mu.Unlock()

	recv = receiver[EData]{ch: make(chan Event[EData]), patterns: map[string]nothing{pattern: 0}}
	b.addReceiver(recv, pattern)
	return recv
}

func (b *Bus[EData]) AddSubscription(pattern string, recv receiver[EData]) {
	b.mu.Lock()
	defer b.mu.Unlock()

	_, ok := recv.patterns[pattern]
	if ok {
		return
	}
	b.addReceiver(recv, pattern)
}

func (b *Bus[EData]) SubscribeCallback(pattern string, cb func(ev Event[EData])) {
	go func(r receiver[EData]) {
		for {
			d, ok := <-r.ch
			if !ok {
				break
			}
			cb(d)
		}
	}(b.Subscribe(pattern))
}

var ErrIllegalParamType = errors.New("illegal parameter type")
var ErrIllegalParamCount = errors.New("illegal parameter count")

// TODO: add UnsubscribeObject

// 1) Searches instance of obj for methods, ending with `{methodSuffix}` or `Async{methodSuffix}`.
// 2) For each method, subscribes it for `{topicPrefix}methodNameWithout{methodSuffix}`
// This method uses reflect package.
func (b *Bus[EData]) SubscribeObject(obj any, handlerSuffix, topicPrefix string) (subCount int, err error) {
	if obj == nil {
		return 0, errors.New("could not register handlers of nil object")
	}
	if handlerSuffix == "" {
		handlerSuffix = "Handler"
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	rVal := reflect.ValueOf(obj)
	if rVal.Kind() != reflect.Pointer && rVal.Kind() != reflect.Interface {
		return 0, errors.New("this method requires instance address, not the instance itself")
	}
	rVal = rVal.Elem()
	objType := rVal.Type()
	numMethods := rVal.NumMethod()

	for i := 0; i < numMethods; i++ {
		m := objType.Method(i)
		evName := m.Name
		if !strings.HasSuffix(evName, handlerSuffix) {
			continue
		}

		numIn := m.Type.NumIn()
		switch numIn {
		case 0:
		case 1:
			t0 := m.Type.In(0)
			etName := reflect.TypeOf(Event[EData]{}).Name()
			if t0.Name() != etName {
				return subCount, fmt.Errorf("%w, expected %s, got %s", ErrIllegalParamType, etName, t0)
			}
		default:
			return subCount, fmt.Errorf("%w, expected 1, got %d", ErrIllegalParamCount, numIn)
		}

		evName = m.Name[:len(m.Name)-len(handlerSuffix)]
		pattern := topicPrefix + evName

		recv := receiver[EData]{ch: make(chan Event[EData]), patterns: map[string]nothing{pattern: 0}}
		b.addReceiver(recv, pattern)
		go func(r receiver[EData], m reflect.Method, numIn int) {
			in := make([]reflect.Value, numIn)
			for {
				d, ok := <-r.ch
				if !ok {
					break
				}
				if numIn == 1 {
					in[0] = reflect.ValueOf(d)
				}
				m.Func.Call(in)
			}
		}(recv, m, numIn)
		subCount++
	}
	return subCount, nil
}

func (b *Bus[EData]) Unsubscribe(topic string, recv receiver[EData]) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.removeReceiver(recv, topic)
}

func (b *Bus[EData]) addReceiver(recv receiver[EData], pattern string) {
	if slices.ContainsFunc(b.recvs[pattern], recv.EqualTo) {
		return
	}
	b.recvs[pattern] = append(b.recvs[pattern], recv)
	recv.addPattern(pattern)
	b.totalRecvs.Add(1)
}

func (b *Bus[EData]) removeReceiver(recv receiver[EData], pattern string) bool {
	patterns := make([]string, 0, 1)
	for rp := range recv.patterns {
		if wildcard.Match(pattern, rp) {
			patterns = append(patterns, rp)
		}
	}
	if len(patterns) == 0 {
		return false
	}

	for _, rp := range patterns {
		l := len(b.recvs[rp])
		b.recvs[rp] = slices.DeleteFunc(b.recvs[rp], recv.EqualTo)
		if l == len(b.recvs[rp]) {
			panic("shouldn't happen")
		}
		recv.removePattern(rp)
		b.totalRecvs.Add(-1)
	}
	if len(recv.patterns) == 0 {
		close(recv.ch)
	}
	return true
}

// Publishes event asynchronously and returns immediately.
func (b *Bus[EData]) Publish(topic string, data EData) {
	dispatch(Event[EData]{Topic: topic, Data: data}, b.getPatternReceivers(topic))
}

// Publishes event synchronously. No goroutine involved.
func (b *Bus[EData]) PublishSync(topic string, data EData) {
	dispatchSync(Event[EData]{Topic: topic, Data: data}, b.getPatternReceivers(topic))
}

// Publishes event asynchronously,
// and waits until all subscribers dispatch it.
// Subscribers must call event.Done() when finished, otherwise PublishWait never returns.
func (b *Bus[EData]) PublishWait(topic string, data EData) {
	receivers := b.getPatternReceivers(topic)
	wg := sync.WaitGroup{}
	wg.Add(len(receivers))

	dispatch(Event[EData]{Topic: topic, Data: data, wg: &wg}, receivers)

	wg.Wait()
}

func (b *Bus[EData]) getPatternReceivers(pattern string) (r []receiver[EData]) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	div := 1 + int(math.Log2(float64(b.totalRecvs.Load())))
	expectedCap := int(b.totalRecvs.Load()) / div
	r = make([]receiver[EData], 0, expectedCap)

	for topic, recvs := range b.recvs {
		if topic == pattern || wildcard.Match(pattern, topic) {
			r = append(r, recvs...)
		}
	}
	log.Printf("receivers, cap wanted %d, got %d", expectedCap, len(r))
	return r
}

func dispatch[EData any](ev Event[EData], recvs []receiver[EData]) {
	go dispatchSync(ev, recvs)
}

func dispatchSync[EData any](ev Event[EData], recvs []receiver[EData]) {
	for _, recv := range recvs {
		recv.ch <- ev
	}
}
