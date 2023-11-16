/*
 * Holds Event data type
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
	"sync"
	"sync/atomic"
)

// Event. Don't pass by reference.
type Event[EData any] struct {
	done  *atomic.Bool
	Data  *EData
	wg    *sync.WaitGroup
	Topic string
}

// If you have code waiting for Publish to process events,
// then Done() can be used to signal that this event instance is done processind earlier than the handler actually returns.
//   - It is not *necessary* to call Done.
//   - It's safe although with no effect to call Done after handler returns (e.g. in goroutine).
//   - It's safe to call Done() multiple times and from different Goroutines but try not to hold event objects
//     longer than their expected lifetime
func (ev *Event[EData]) Done() {
	if ev.done.CompareAndSwap(false, true) {
		ev.wg.Done()
	}
}
