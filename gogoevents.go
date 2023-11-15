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
	"reflect"
	"strings"
	"sync"

	"github.com/amanofbits/gogoevents/internal/wildcard"
)

type Bus[EData any] struct {
	recvs *receiverCollection[EData]
}

func NewUntyped() *Bus[any] {
	return New[any]()
}

func New[EData any]() *Bus[EData] {
	return &Bus[EData]{
		recvs: newReceiverCollection[EData](),
	}
}

func (b *Bus[EData]) Close() error {
	b.recvs.close()
	return nil
}

func (b *Bus[EData]) Subscribe(pattern string) (recv Receiver[EData]) {
	recv = newReceiver[EData](pattern)
	b.recvs.add(recv)
	return recv
}

// Syntactic sugar for subscribing callback function to topic pattern.
// Returns receiver that can be used for unsubscribing.
func (b *Bus[EData]) SubscribeCallback(pattern string, cb func(ev Event[EData])) Receiver[EData] {
	recv := b.Subscribe(pattern)
	go func(r Receiver[EData]) {
		for {
			d, ok := <-r._ch
			if !ok {
				break
			}
			cb(d)
		}
	}(recv)
	return recv
}

var ErrIllegalParamType = errors.New("illegal parameter type")
var ErrIllegalParamCount = errors.New("illegal parameter count")
var ErrReturnValuesPresent = errors.New("return values present and not allowed")

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

		if m.Type.NumOut() > 0 {
			return subCount, fmt.Errorf("%s: %w", m.Name, ErrReturnValuesPresent)
		}
		numIn := m.Type.NumIn()
		switch numIn {
		case 0:
		case 1:
			t0 := m.Type.In(0)
			etName := reflect.TypeOf(Event[EData]{}).Name()
			if t0.Name() != etName {
				return subCount, fmt.Errorf("%s: %w, expected %s, got %s", m.Name, ErrIllegalParamType, etName, t0)
			}
		default:
			return subCount, fmt.Errorf("%s: %w, expected 1, got %d", m.Name, ErrIllegalParamCount, numIn)
		}

		evName = m.Name[:len(m.Name)-len(handlerSuffix)]
		pattern := topicPrefix + evName

		recv := newReceiver[EData](pattern)
		b.recvs.add(recv)

		go func(r Receiver[EData], m reflect.Method, numIn int) {
			in := make([]reflect.Value, numIn)
			for {
				d, ok := <-r._ch
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

// Removes specified receiver from subscriptions.
// If optional patterns are specified, unsubscribes only from these patterns.
// If receiver has no more topics after this operation, its channel gets closed before return.
// Patterns here does not use wildcard matching and are matched as is.
func (b *Bus[EData]) Unsubscribe(recv Receiver[EData]) bool {
	return b.recvs.remove(recv)
}

var ErrIllegalWildcard = errors.New("wildcards not allowed in topic")

// Publishes event asynchronously and returns immediately.
// Returns `ErrIllegalWildcard` if wildcard is found in the `topic`
func (b *Bus[EData]) Publish(topic string, data EData) error {
	if wci := wildcard.Index(topic); wci >= 0 {
		return fmt.Errorf("%s, at %d: %w", topic, wci, ErrIllegalWildcard)
	}
	b.dispatch(Event[EData]{Topic: topic, Data: data}, b.recvs.get(topic))
	return nil
}

// Publishes event and waits until all subscribers dispatch it.
// Subscribers must call event.Done() when finished, otherwise PublishWait never returns.
// Returns `ErrIllegalWildcard` if wildcard is found in the `topic`
func (b *Bus[EData]) PublishSync(topic string, data EData) error {
	if wci := wildcard.Index(topic); wci >= 0 {
		return fmt.Errorf("%s, at %d: %w", topic, wci, ErrIllegalWildcard)
	}
	recvs := b.recvs.get(topic)
	wg := sync.WaitGroup{}
	wg.Add(len(recvs))

	b.dispatch(Event[EData]{Topic: topic, Data: data, wg: &wg}, recvs)

	wg.Wait()
	return nil
}

func (b *Bus[EData]) dispatch(ev Event[EData], recvs []Receiver[EData]) {
	for _, recv := range recvs {
		b.recvs.recvOps <- pushOp[EData]{recv: recv, ev: ev}
	}
}
