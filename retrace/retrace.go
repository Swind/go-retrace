package retrace

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"unicode"
)

type Retrace struct {
	RegularExpression  string
	RegularExpression2 string
	AllClassNames      bool
	Verbose            bool
	MappingFileReader  io.Reader
}

// For example: "com.example.Foo.bar"
var REGULAR_EXPRESSION_CLASS_METHOD = `%c\.%m`

// For example:
// "(Foo.java:123:0) ~[0]"
// "()(Foo.java:123:0)"     (DGD-1732, unknown origin, possibly Sentry)
// or no source line info   (DGD-1732, Sentry)
var REGULAR_EXPRESSION_SOURCE_LINE = `(?:\(\))?(?:\((?:%s)?(?::?%l)?(?::\d+)?\))?\s*(?:~\[.*\])?`

// For example: "at o.afc.b + 45(:45)"
// Might be present in recent stacktraces accessible from crashlytics.
var REGULAR_EXPRESSION_OPTIONAL_SOURCE_LINE_INFO = `(?:\+\s+[0-9]+)?`

// For example: "    at com.example.Foo.bar(Foo.java:123:0) ~[0]"
var REGULAR_EXPRESSION_AT = `.*?\bat\s+` + REGULAR_EXPRESSION_CLASS_METHOD + `\s*` + REGULAR_EXPRESSION_OPTIONAL_SOURCE_LINE_INFO + REGULAR_EXPRESSION_SOURCE_LINE

// For example: "java.lang.ClassCastException: com.example.Foo cannot be cast to com.example.Bar"
// Every line can only have a single matched class, so we try to avoid
// longer non-obfuscated class names.
var REGULAR_EXPRESSION_CAST1 = `.*?\bjava\.lang\.ClassCastException: %c cannot be cast to .{5,}`
var REGULAR_EXPRESSION_CAST2 = `.*?\bjava\.lang\.ClassCastException: .* cannot be cast to %c`

// For example: "java.lang.NullPointerException: Attempt to read from field 'java.lang.String com.example.Foo.bar' on a null object reference"
var REGULAR_EXPRESSION_NULL_FIELD_READ = `.*?\bjava\.lang\.NullPointerException: Attempt to read from field '%t %c\.%f' on a null object reference`

// For example: "java.lang.NullPointerException: Attempt to write to field 'java.lang.String com.example.Foo.bar' on a null object reference"
var REGULAR_EXPRESSION_NULL_FIELD_WRITE = `.*?\bjava\.lang\.NullPointerException: Attempt to write to field '%t %c\.%f' on a null object reference`

// For example: "java.lang.NullPointerException: Attempt to invoke virtual method 'void com.example.Foo.bar(int,boolean)' on a null object reference"
var REGULAR_EXPRESSION_NULL_METHOD = `.*?\bjava\.lang\.NullPointerException: Attempt to invoke (?:virtual|interface) method '%t %c\.%m\(%a\)' on a null object reference`

// For example: "Something: com.example.FooException: something"
var REGULAR_EXPRESSION_THROW = `(?:.*?[:\"]\\s+)?%c(?::.*)?`

// For example: java.lang.NullPointerException: Cannot invoke "com.example.Foo.bar.foo(int)" because the return value of "com.example.Foo.bar.foo2()" is null
var REGULAR_EXPRESSION_RETURN_VALUE_NULL1 = `.*?\bjava\.lang\.NullPointerException: Cannot invoke \".*\" because the return value of \"%c\.%m\(%a\)\" is null`
var REGULAR_EXPRESSION_RETURN_VALUE_NULL2 = `.*?\bjava\.lang\.NullPointerException: Cannot invoke \"%c\.%m\(%a\)\" because the return value of \".*\" is null`

// For example: Cannot invoke "java.net.ServerSocket.close()" because "com.example.Foo.bar" is null
var REGULAR_EXPRESSION_BECAUSE_IS_NULL = `.*?\bbecause \"%c\.%f\" is null`

// The overall regular expression for a line in the stack trace.
var REGULAR_EXPRESSION = "(?:" + REGULAR_EXPRESSION_AT + ")|" +
	"(?:" + REGULAR_EXPRESSION_CAST1 + ")|" +
	"(?:" + REGULAR_EXPRESSION_CAST2 + ")|" +
	"(?:" + REGULAR_EXPRESSION_NULL_FIELD_READ + ")|" +
	"(?:" + REGULAR_EXPRESSION_NULL_FIELD_WRITE + ")|" +
	"(?:" + REGULAR_EXPRESSION_NULL_METHOD + ")|" +
	"(?:" + REGULAR_EXPRESSION_RETURN_VALUE_NULL1 + ")|" +
	"(?:" + REGULAR_EXPRESSION_BECAUSE_IS_NULL + ")|" +
	"(?:" + REGULAR_EXPRESSION_THROW + ")"

// DIRTY FIX:
// We need to call another regex because Java 16 stacktrace may have multiple methods in the same line.
// For Example: java.lang.NullPointerException: Cannot invoke "dev.lone.itemsadder.Core.f.a.b.b.b.c.a(org.bukkit.Location, boolean)" because the return value of "dev.lone.itemsadder.Core.f.a.b.b.b.c.a()" is null
// TODO: Make this stuff less hacky.
var REGULAR_EXPRESSION2 = "(?:" + REGULAR_EXPRESSION_RETURN_VALUE_NULL2 + ")"

func NewRetrace(mappingFileReader io.Reader) *Retrace {
	retrace := Retrace{}

	retrace.RegularExpression = REGULAR_EXPRESSION
	retrace.RegularExpression2 = REGULAR_EXPRESSION2
	retrace.AllClassNames = false
	retrace.Verbose = false
	retrace.MappingFileReader = mappingFileReader

	return &retrace
}

func (r *Retrace) Retrace(reader io.Reader, writer io.Writer) {
	bufWriter := bufio.NewWriter(writer)

	// create a pattern for stack frames
	pattern1 := NewFramePattern(r.RegularExpression, r.Verbose)
	pattern2 := NewFramePattern(r.RegularExpression2, r.Verbose)

	mapper := NewFrameRemapper()

	// Read the mapping file
	mappingReader := NewMappingReader(r.MappingFileReader)
	mappingReader.Pump(mapper)

	// Read and process the lines of the stack trace.
	bufReader := bufio.NewReader(reader)
	for {
		obfuscatedLine, err := bufReader.ReadString('\n')
		if err != nil {
			break
		}

		obfuscatedFrame1 := pattern1.Parse(obfuscatedLine)
		obfuscatedFrame2 := pattern2.Parse(obfuscatedLine)

		deobf := r.handle(&obfuscatedFrame1, mapper, pattern1, &obfuscatedLine)
		// DIRTY FIX:
		// I have to execute it two times because recent Java stacktraces may have multiple fields/methods in the same line.
		// For example: java.lang.NullPointerException: Cannot invoke "com.example.Foo.bar.foo(int)" because the return value of "com.example.Foo.bar.foo2()" is null
		deobf = r.handle(&obfuscatedFrame2, mapper, pattern2, &deobf)

		bufWriter.WriteString(deobf)
	}

	bufWriter.Flush()
}

func (r *Retrace) handle(obfuscatedFrame *FrameInfo, mapper *FrameRemapper, pattern *FramePattern, obfuscatedLine *string) string {
	result := bytes.NewBufferString("")
	if obfuscatedFrame != nil {
		// Transform the obfuscated frame back to one or more original frames.
		retracedFrames := mapper.Transform(obfuscatedFrame)

		var previousLine *string = nil

		for _, retracedFrame := range retracedFrames {
			retracedLine := pattern.Format(*obfuscatedLine, retracedFrame)

			// Clear the common first part of ambiguous alternative
			// retraced lines, to present a cleaner list of alternatives.
			var trimmedLine = retracedLine
			if previousLine != nil && obfuscatedFrame.LineNumber == 0 {
				trimmedLine = r.Trim(&retracedLine, previousLine)
			}

			// Print out the retraced line
			if len(trimmedLine) != 0 {
				if r.AllClassNames {
					trimmedLine = r.Deobfuscate(&trimmedLine, mapper)
				}
				result.WriteString(trimmedLine)
			}

			previousLine = &retracedLine
		}
	} else {
		if r.AllClassNames {
			result.WriteString(r.Deobfuscate(obfuscatedLine, mapper))
		} else {
			result.WriteString(*obfuscatedLine)
		}
	}

	return result.String()
}

/**
 * Returns the first given string, with any leading characters that it has
 * in common with the second string replaced by spaces.
 */
func (r *Retrace) Trim(string1 *string, string2 *string) string {
	buffer := bytes.NewBufferString("")

	// Find the common part.
	trimEnd := r.FirstNonCommonIndex(string1, string2)

	// Clear the common characters
	for i := 0; i < trimEnd; i++ {
		buffer.WriteString(" ")
	}
	buffer.WriteString((*string1)[trimEnd:])

	return buffer.String()
}

func (r *Retrace) FirstNonCommonIndex(string1 *string, string2 *string) int {
	var i int
	for i = 0; i < len(*string1) && i < len(*string2); i++ {
		if (*string1)[i] != (*string2)[i] {
			return i
		}
	}
	return i
}

func deobfuscateFieldsFunc(c rune) bool {
	return unicode.IsSpace(c) ||
		c == '(' || c == ')' ||
		c == '<' || c == '>' ||
		c == '[' || c == ']' ||
		c == '{' || c == '}' ||
		c == ';' || c == ':' || c == ',' ||
		c == '\'' || c == '"' ||
		c == '/' || c == '\\'
}

func (r *Retrace) Deobfuscate(line *string, mapper *FrameRemapper) string {
	var buff strings.Builder

	// Try to deobfuscate any token encountered in the line.
	tokens := FieldsFuncWithDelims(*line, deobfuscateFieldsFunc)
	for _, token := range tokens {
		// Try to deobfuscate the token.
		if len(token) == 1 && deobfuscateFieldsFunc(rune(token[0])) {
			// Don't try to deobfuscate delimiters.
			buff.WriteString(token)
		} else {
			buff.WriteString(mapper.GetOriginalClassName(token))
		}
	}
	return buff.String()
}
