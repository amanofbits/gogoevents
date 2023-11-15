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
	"testing"
	"time"

	"github.com/amanofbits/gogoevents"
)

func Benchmark(b *testing.B) {
	b.StopTimer()

	eb := gogoevents.NewUntyped()
	recvs := make([]gogoevents.Receiver[any], 10)
	for i := 0; i < len(recvs); i++ {
		recvs[i] = eb.Subscribe("t*")
	}
	//
	for _, recv := range recvs {
		go func(recv gogoevents.Receiver[any]) {
			for range recv.Ch() {
			}
		}(recv)
	}

	for i := 0; i < b.N; i++ {
		b.StartTimer()
		eb.Publish("test", nil)
		b.StopTimer()
	}
	eb.Close()
}

func TestSmoke(t *testing.T) {
	eb := gogoevents.NewUntyped()
	r := eb.Subscribe("test")

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
	case v, ok := <-r.Ch():
		if !ok {
			t.Fatal("no event, channel closed")
		}
		v.Done()
	case <-cl:
		t.Fatal("no event received")
	}
}

func TestAddSubscription(t *testing.T) {
	eb := gogoevents.NewUntyped()
	r := eb.Subscribe("test")
	eb.AddSubscription("test2", r)

	eb.Publish("test2", nil)

	cl := make(chan any)
	go func() {
		time.Sleep(time.Second)
		close(cl)
	}()

	select {
	case v, ok := <-r.Ch():
		if !ok {
			t.Fatal("no event, channel closed")
		}
		if v.Topic != "test2" {
			t.Fatalf("wrong topic. Wanted %s, got %s", "test2", v.Topic)
		}
		v.Done()
	case <-cl:
		t.Fatal("no event received")
	}
}

func TestSubscribeCallback(t *testing.T) {
	eb := gogoevents.NewUntyped()
	fired := false
	eb.SubscribeCallback("test", func(ev gogoevents.Event[any]) {
		if ev.Topic != "test" {
			t.Fatal("illegal topic", ev.Topic)
		}
		if ev.Data != nil {
			t.Fatal("illegal data", ev.Data)
		}
		fired = true
		ev.Done()
	})

	eb.Publish("test", nil)

	time.Sleep(100 * time.Millisecond)
	if !fired {
		t.Fatal("not fired")
	}
}
