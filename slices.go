/*
 * Holds slice helpers
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

// Modified copy from slices.BinarySearchFunc, Go 1.21.4, with element index.
// Why not include item's index into comparison function?..
//
// The slice must be sorted in increasing order, where "increasing"
// is defined by cmp. cmp should return 0 if the slice element matches
// the target, a negative number if the slice element precedes the target,
// or a positive number if the slice element follows the target.
// cmp must implement the same ordering as the slice, such that if
// cmp(a, t) < 0 and cmp(b, t) >= 0, then a must precede b in the slice.
func binarySearchFunc[S ~[]E, E, T any](x S, target T, cmp func(E, T, int) int) (int, bool) {
	n := len(x)
	// Define cmp(x[-1], target) < 0 and cmp(x[n], target) >= 0 .
	// Invariant: cmp(x[i - 1], target) < 0, cmp(x[j], target) >= 0.
	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1) // avoid overflow when computing h
		// i ≤ h < j
		if cmp(x[h], target, h) < 0 {
			i = h + 1 // preserves cmp(x[i - 1], target) < 0
		} else {
			j = h // preserves cmp(x[j], target) >= 0
		}
	}
	// i == j, cmp(x[i-1], target) < 0, and cmp(x[j], target) (= cmp(x[i], target)) >= 0  =>  answer is i.
	return i, i < n && cmp(x[i], target, i) == 0
}

// Modified copy from slices.Insert, 1.21.4. Simplified for Subscribe use case.
// Shifts slice to the right, starting at idx
func shift[S ~[]E, E any](src S, idx int) S {
	srcLen := len(src)
	if srcLen+1 > cap(src) {
		src = append(src[:cap(src)], *new(E))
	}
	src = src[:srcLen+1]

	if idx == srcLen {
		return src
	}

	copy(src[idx+1:srcLen+1], src[idx:srcLen])
	return src
}
