package retrace

func FieldsFuncWithDelims(s string, f func(rune) bool) []string {
	// A span is used to record a slice of s of the form s[start:end].
	// The start index is inclusive and the end index is exclusive.
	type span struct {
		start int
		end   int
	}
	spans := make([]span, 0, 32)

	// Find the field start and end indices.
	// Doing this in a separate pass (rather than slicing the string s
	// and collecting the result substrings right away) is significantly
	// more efficient, possibly due to cache effects.
	start := -1 // valid span start if >= 0
	for end, rune := range s {
		if f(rune) {
			if start >= 0 {
				spans = append(spans, span{start, end})
				spans = append(spans, span{end, end + 1}) // delimiter is also a field
				// Set start to a negative value.
				// Note: using -1 here consistently and reproducibly
				// slows down this code by a several percent on amd64.
				start = ^start
			}
		} else {
			if start < 0 {
				start = end
			}
		}
	}

	// Last field might end at EOF.
	if start >= 0 {
		spans = append(spans, span{start, len(s)})
	}

	// Create strings from recorded field indices.
	a := make([]string, len(spans))
	for i, span := range spans {
		a[i] = s[span.start:span.end]
	}

	return a
}
