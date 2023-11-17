/*
 * Holds wildcard tests
 *
 * Original code copyright:
 *
 * Copyright (c) 2023 Iglou.eu <contact@iglou.eu>
 * Copyright (c) 2023 Adrien Kara <adrien@iglou.eu>
 *
 * This file is part of gogoevents.
 * modifications copyright:
 * © 2023 amanofbits
 *
 * gogoevents is free software: you can redistribute it and/or modify
 * it under the terms of the BSD 3-Clause License as published by the
 * University of California. See the `LICENSE` file for more details.
 *
 * You should have received a copy of the BSD 3-Clause License along with
 * gogoevents. If not, see <https://opensource.org/licenses/BSD-3-Clause>.
 */

package wildcard_test

import (
	"testing"

	"github.com/amanofbits/gogoevents/internal/wildcard"
)

func TestNormalize(t *testing.T) {
	s := []string{
		"",
		"test",
		"t",
		"**",
		"*big*test*",
		"**big******test***2*2",
	}
	ans := []string{
		"",
		"test",
		"t",
		"*",
		"*big*test*",
		"*big*test*2*2",
	}

	for i := range s {
		a := wildcard.Normalize(s[i])
		if ans[i] != a {
			t.Fatalf("expected %s, got %s", ans[i], a)
		}
	}
}
