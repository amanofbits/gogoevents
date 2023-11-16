/*
 * Holds unique id generator
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
	"fmt"
	"math"
	"sync/atomic"
)

var (
	// Don't use this directly!
	_id atomic.Uint64
)

const _safeIdMargin = math.MaxUint64 - 10 // 10 is arbitrary.

func newUniqueId() uint64 {
	id := _id.Add(1)
	if id > _safeIdMargin { // Not sure if someone will ever reach here, but fail is better than UB.
		panic(fmt.Sprintf("incremental subscriber id overflow. Value %d", id))
	}
	return id
}
