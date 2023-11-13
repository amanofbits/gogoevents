/*
 * Holds event receiver
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

type nothing = uint8

type receiver[EData any] struct {
	ch       chan Event[EData]
	patterns map[string]nothing
}

func (r receiver[EData]) Ch() chan Event[EData] { return r.ch }

func (r *receiver[EData]) addPattern(p string) {
	r.patterns[p] = 0
}

func (r *receiver[EData]) removePattern(p string) {
	delete(r.patterns, p)
}

func (this receiver[EData]) EqualTo(r receiver[EData]) bool { return this.ch == r.ch }
