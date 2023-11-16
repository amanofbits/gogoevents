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

package gogoevents_test

import (
	"log"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/amanofbits/gogoevents"
)

func TestSmoke(t *testing.T) {
	log.Println(
		unsafe.Sizeof(func(ev gogoevents.Event[complex128]) {}),
	)
	eb := gogoevents.NewUntyped()
	ch := make(chan any)
	eb.Subscribe("test", func(ev gogoevents.Event[any]) {
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

	eb := gogoevents.NewUntyped()
	subs := make([]gogoevents.Subscriber[any], subsCount)

	eventsGot := atomic.Int32{}

	for i := 0; i < len(subs); i++ {
		subs[i] = eb.Subscribe(topic[:i%(len(topic)-1)+1]+"*", func(ev gogoevents.Event[any]) {
			eventsGot.Add(1)
		})
	}

	for i := 0; i < 10000; i++ {
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

func BenchmarkEventPublish(b *testing.B) {
	topic := "testevent"
	subsCount := int32(1000)

	eb := gogoevents.NewUntyped()
	subs := make([]gogoevents.Subscriber[any], subsCount)
	eventsGot := atomic.Int32{}

	for i := 0; i < len(subs); i++ {
		subs[i] = eb.Subscribe(topic[:i%(len(topic)-1)+1]+"*", func(ev gogoevents.Event[any]) {
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
	topic := "testevent"
	subsCount := 1000

	subs := make([]gogoevents.Subscriber[any], subsCount)
	t := ""
	h := func(ev gogoevents.Event[any]) {}

	b.StopTimer()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		eb := gogoevents.NewUntyped()
		b.StartTimer() // not very accurate but it takes forever to start/stop timer within the loop
		for i := 0; i < subsCount; i++ {
			t = topic[:i%(len(topic)-1)+1] + "*"
			subs[i] = eb.Subscribe(t, h)
		}
		b.StopTimer()
		eb.Close()
	}

}
