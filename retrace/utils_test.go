package retrace

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldsFuncWithDelims(t *testing.T) {
	s := "java.lang.NullPointerException: Attempt to invoke virtual method 'java.lang.String java.lang.Object.toString()' on a null object reference"
	expected := []string{
		"java.lang.NullPointerException:",
		" ",
		"Attempt",
		" ",
		"to",
		" ",
		"invoke",
		" ",
		"virtual",
		" ",
		"method",
		" ",
		"'java.lang.String",
		" ",
		"java.lang.Object.toString()'",
		" ",
		"on",
		" ",
		"a",
		" ",
		"null",
		" ",
		"object",
		" ",
		"reference",
	}

	actual := FieldsFuncWithDelims(s, func(r rune) bool {
		return r == ' '
	})

	assert.Equal(t, expected, actual)
}
