/*
 * Holds tests for event bus.
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
	"math"
	"math/rand"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestSmoke(t *testing.T) {
	eb := NewUntyped()
	ch := make(chan any)
	eb.Subscribe("test", func(ev Event[any]) {
		ch <- ev
	})

	go func() {
		time.Sleep(100 * time.Millisecond)
		eb.Publish("test", nil)
	}()

	cl := make(chan any)
	go func() {
		time.Sleep(time.Second)
		close(cl)
	}()

	select {
	case _, ok := <-ch:
		if !ok {
			t.Fatal("no event, channel closed")
		}
	case <-cl:
		t.Fatal("no event received")
	}
}

func TestAllEventHandlersAreWaited(t *testing.T) {
	topic := "testevent"
	subsCount := int32(1000)

	eb := NewUntyped()
	subs := make([]Subscriber[any], subsCount)

	eventsGot := atomic.Int32{}

	for i := 0; i < len(subs); i++ {
		subs[i] = eb.Subscribe(topic[:i%(len(topic)-1)+1]+"*", func(ev Event[any]) {
			eventsGot.Add(1)
		})
	}

	for i := 0; i < 1000; i++ {
		eventsGot.Store(0)
		wg, err := eb.Publish("testevent", strconv.Itoa(i))
		if err != nil {
			t.Fatal("got error", err)
		}
		wg.Wait()
		got := eventsGot.Load()
		if got != subsCount {
			t.Fatalf("bad event count after wait. Expected %d, got %d", subsCount, got)
		}
	}
	eb.Close()
}

func TestUnsubscribe(t *testing.T) {
	subsCount := 10000

	eb := NewUntyped()

	subs := make([]Subscriber[any], subsCount)

	for i := 0; i < len(subs); i++ {
		subs[i] = eb.Subscribe(strconv.FormatUint(rand.Uint64(), 16), func(ev Event[any]) {
		})
	}

	if eb.TotalSubscribers() != subsCount {
		t.Fatalf("Subscribe failed. Wanted %d, got %d subscribers", subsCount, eb.TotalSubscribers())
	}
	sub := subs[rand.Int()%subsCount]

	if len(subs) != subsCount {
		t.Fatalf("illegal subscribers count. Expected %d, got %d", subsCount, len(subs))
	}
	res := eb.Unsubscribe(sub)
	if eb.TotalSubscribers() != subsCount-1 {
		t.Fatalf("subs count hasn't changed")
	}
	if !res {
		t.Fatalf("subs count changed but unsubscribe returned false")
	}
	if len(eb.subs) != len(eb.patterns) {
		t.Fatalf("subs length %d != patterns length %d. Forgot to remove sub's pattern?", len(eb.subs), len(eb.patterns))
	}
}

func TestUnsubscribeLast(t *testing.T) {
	subsCount := 10

	eb := NewUntyped()
	subs := make([]Subscriber[any], subsCount)

	for i := 0; i < len(subs); i++ {
		subs[i] = eb.Subscribe(strconv.FormatUint(rand.Uint64(), 16), func(ev Event[any]) {
		})
	}

	if eb.TotalSubscribers() != subsCount {
		t.Fatalf("Subscribe failed. Wanted %d, got %d subscribers", subsCount, eb.TotalSubscribers())
	}
	if len(subs) != subsCount {
		t.Fatalf("illegal subscribers count. Expected %d, got %d", subsCount, len(subs))
	}

	lastSub := eb.subs[len(eb.subs)-1]
	res := eb.Unsubscribe(lastSub)
	if eb.TotalSubscribers() != subsCount-1 {
		t.Fatalf("subs count hasn't changed")
	}
	if !res {
		t.Fatalf("subs count changed but unsubscribe returned false")
	}
	if len(eb.subs) != len(eb.patterns) {
		t.Fatalf("subs length %d != patterns length %d. Forgot to remove sub's pattern?", len(eb.subs), len(eb.patterns))
	}
}

func BenchmarkUnsubscribe(b *testing.B) {
	b.StopTimer()
	b.ResetTimer()

	eb := NewUntyped()

	f := func(ev Event[any]) {}
	subsCount := 10000
	subs := []Subscriber[any]{}
	cnt := 0

	for i := 0; i < b.N; i++ {
		if cnt == 0 {
			cnt = subsCount
			for i := 0; i < cnt; i++ {
				subs = append(subs, eb.Subscribe(strconv.FormatUint(rand.Uint64(), 16), f))
			}
			if eb.TotalSubscribers() != cnt {
				b.Fatalf("Subscribe failed. Wanted %d, got %d subscribers", cnt, eb.TotalSubscribers())
			}
		}

		idx := rand.Int() % cnt
		sub := subs[idx]
		subs = append(subs[:idx], subs[idx+1:]...)

		b.StartTimer()
		res := eb.Unsubscribe(sub)
		b.StopTimer()
		if eb.TotalSubscribers() != cnt-1 {
			b.Fatalf("subs count hasn't changed")
		}
		cnt--
		if !res {
			b.Fatalf("subs count changed but unsubscribe returned false")
		}
	}
}

func BenchmarkEventPublish(b *testing.B) {
	topic := "testevent"
	subsCount := int32(1000)

	eb := NewUntyped()
	subs := make([]Subscriber[any], subsCount)
	eventsGot := atomic.Int32{}

	for i := 0; i < len(subs); i++ {
		subs[i] = eb.Subscribe(topic[:i%(len(topic)-1)+1]+"*", func(ev Event[any]) {
			eventsGot.Add(1)
		})
	}

	b.StopTimer()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eventsGot.Store(0)
		b.StartTimer()
		wg, err := eb.Publish("testevent", strconv.Itoa(i))
		if err != nil {
			b.Fatal("got error", err)
		}
		wg.Wait()
		b.StopTimer()
		got := eventsGot.Load()
		if got != subsCount {
			b.Fatalf("bad event count after wait. Wanted %d, got %d", subsCount, got)
		}
	}

	eb.Close()
}

func BenchmarkSubscribe(b *testing.B) {
	b.StopTimer()
	b.ResetTimer()

	// check values below to match iteration size below
	topic := "testevent23"
	subsCount := int(math.Pow10(3))

	topics := []string{}
	for i := 0; i < len(topic)-1; i++ {
		topics = append(topics, topic[:i+1]+"*")
	}

	subs := make([]Subscriber[any], subsCount)
	h := func(ev Event[any]) {}

	for i := 0; i < b.N; i++ {
		eb := NewUntyped()
		for i := 0; i < subsCount; i += 10 {
			j := i % len(topics)
			b.StartTimer() // not very accurate but it takes forever to start/stop timer every time within the loop
			subs[i+0] = eb.Subscribe(topics[j+0], h)
			subs[i+1] = eb.Subscribe(topics[j+1], h)
			subs[i+2] = eb.Subscribe(topics[j+2], h)
			subs[i+3] = eb.Subscribe(topics[j+3], h)
			subs[i+4] = eb.Subscribe(topics[j+4], h)
			subs[i+5] = eb.Subscribe(topics[j+5], h)
			subs[i+6] = eb.Subscribe(topics[j+6], h)
			subs[i+7] = eb.Subscribe(topics[j+7], h)
			subs[i+8] = eb.Subscribe(topics[j+8], h)
			subs[i+9] = eb.Subscribe(topics[j+9], h)
			b.StopTimer()
		}
		eb.Close()
	}
}
