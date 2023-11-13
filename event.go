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

import "sync"

type Event[EData any] struct {
	Topic string
	Data  EData
	wg    *sync.WaitGroup
}

// This needs to e called when handling is completed.
// In case of PublishWait events, call to Done() is essential for method to return.
// In case of simple Publish, Done() is a no-op.
func (ev *Event[EData]) Done() {
	if ev.wg != nil {
		ev.Done()
	}
}
