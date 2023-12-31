/*
 * Wildcard string matching algorithm, taken and slightly modified from:
 * https://github.com/IGLOU-EU/go-wildcard/blob/2f93770ccbe7d1f3e102221d88ade4c0ecca52be/wildcard.go
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

package wildcard

// Match returns true if the pattern matches the s string.
// The pattern can contain the wildcard character '*'.
func Match(pattern, s string) bool {
	if pattern == "" {
		return s == pattern
	}
	if pattern == "*" || s == pattern {
		return true
	}

	return match(pattern, s)
}

func match(pattern, s string) bool {
	var patternIndex, sIndex, lastStar int
	patternLen := len(pattern)
	sLen := len(s)
	star := -1

Loop:
	if sIndex >= sLen {
		goto checkPattern
	}

	if patternIndex >= patternLen {
		if star != -1 {
			patternIndex = star + 1
			lastStar++
			sIndex = lastStar
			goto Loop
		}
		return false
	}
	switch pattern[patternIndex] {
	case '*':
		// '*' matches zero or more characters. Store its position and increment the pattern index.
		star = patternIndex
		lastStar = sIndex
		patternIndex++
		goto Loop
	default:
		// If the characters don't match, check if there was a previous '*' to backtrack.
		if pattern[patternIndex] != s[sIndex] {
			if star != -1 {
				patternIndex = star + 1
				lastStar++
				sIndex = lastStar
				goto Loop
			}

			return false
		}
	}

	patternIndex++
	sIndex++
	goto Loop

	// Check if the remaining pattern characters is '*', which can match the end of the string.
checkPattern:
	if patternIndex < patternLen {
		if pattern[patternIndex] == '*' {
			patternIndex++
			goto checkPattern
		}
	}

	return patternIndex == patternLen
}

// Returns index of first wildcard symbol, or -1 if not found.
func Index(s string) int {
	for i, r := range []rune(s) {
		if r == '*' {
			return i
		}
	}
	return -1
}

// Removes redundand consecutive wildcards and returns modified string
func Normalize(s string) string {
	if len(s) < 2 {
		return s
	}
	sr := []rune(s)
	for i, j := 1, -1; i < len(sr); i++ {
		if sr[i] != '*' || sr[i-1] != '*' {
			continue
		}
		for j = i + 1; j < len(sr) && sr[j] == '*'; {
			j++
		}
		sr = append(sr[:i], sr[j:]...)
		i--
	}
	return string(sr)
}
