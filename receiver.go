/*
 * Holds event receiver and operations that are applicable to them
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

// Remember to use receiver ops goroutine and channel to perform operations on receivers.

type Receiver[EData any] struct {
	_ch      chan Event[EData]
	patterns map[string]void
}

func newReceiver[EData any]() Receiver[EData] {
	return Receiver[EData]{
		_ch:      make(chan Event[EData]),
		patterns: make(map[string]struct{}, 1),
	}
}

func (r Receiver[EData]) Ch() <-chan Event[EData] { return r._ch }

func (this Receiver[EData]) EqualTo(r Receiver[EData]) bool { return this._ch == r._ch }

// Operation that can be applied to receivers.
// Processed sequentially in a goroutine.
// This is done to prevent situations when e.g. event is being sent to the channel that was just closed.
type recvOp[EData any] interface {
	do()
}

type pushOp[EData any] struct {
	recv Receiver[EData]
	ev   Event[EData]
}

func (op pushOp[EData]) do() {
	op.recv._ch <- op.ev
}

type closeOp[EData any] Receiver[EData]

func (op closeOp[EData]) do() {
	close(Receiver[EData](op)._ch)
}
