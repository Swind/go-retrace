package retrace

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
)

// Reference: https://github.com/Guardsquare/proguard/blob/0344c58b3d43799ce203737eea3fd1b58ca701ad/retrace/src/proguard/retrace/FramePattern.java
// Reference: https://github.com/Guardsquare/proguard/blob/b0104ecd96ed0577b66ba28d10f7f6d2e748e8d4/retrace/src/proguard/retrace/ReTrace.java

const REGEX_CLASS = `(?:[^\s\":./()]+\.)*[^\s":./()]+`
const REGEX_CLASS_SLASH = `(?:[^\s\":./()]+/)*[^\s":./()]+`
const REGEX_SOURCE_FILE = `(?:[^:()\d][^:()]*)?`
const REGEX_LINE_NUMBER = `-?\b\d+\b`
const REGEX_MEMBER = `<?[^\s\":./()]+>?`

var REGEX_TYPE = REGEX_CLASS + `(?:\[\])*`
var REGEX_ARGUMENTS = `(?:` + REGEX_TYPE + `(?:\s*,\s*` + REGEX_TYPE + ")*)?"

/**
 * This class can parse and format lines that represent stack frames
 * matching a given regular expression.
 */
type FramePattern struct {
	RegularExpression string

	ExpressionTypes     [32]string
	ExpressionTypeCount int
	Pattern             regexp.Regexp
	Verbose             bool
}

func NewFramePattern(regularExpression string, verbose bool) *FramePattern {
	framePattern := FramePattern{
		RegularExpression: regularExpression,
		Verbose:           verbose,
	}

	buffer := bytes.NewBufferString("")

	var expressionTypeCount = 0
	var index = 0

	for {
		nextIndex := strings.Index(regularExpression[index:], "%")
		if nextIndex < 0 ||
			nextIndex == len(regularExpression)-1 ||
			expressionTypeCount == len(framePattern.ExpressionTypes) {
			break
		}
		nextIndex += index

		// Copy a literal piece of the input line.
		buffer.WriteString(regularExpression[index:nextIndex])
		buffer.WriteString("(")

		expressionType := regularExpression[nextIndex+1 : nextIndex+2]
		switch expressionType {
		case "c":
			buffer.WriteString(REGEX_CLASS)
		case "C":
			buffer.WriteString(REGEX_CLASS_SLASH)
		case "s":
			buffer.WriteString(REGEX_SOURCE_FILE)
		case "l":
			buffer.WriteString(REGEX_LINE_NUMBER)
		case "t":
			buffer.WriteString(REGEX_TYPE)
		case "f":
			buffer.WriteString(REGEX_MEMBER)
		case "m":
			buffer.WriteString(REGEX_MEMBER)
		case "a":
			buffer.WriteString(REGEX_ARGUMENTS)
		}

		buffer.WriteString(")")
		expressionTypeCount++
		framePattern.ExpressionTypes[expressionTypeCount] = expressionType
		index = nextIndex + 2
	}

	// Copy the last literal piece of the input line.
	buffer.WriteString(regularExpression[index:])

	framePattern.ExpressionTypeCount = expressionTypeCount
	framePattern.Pattern = *regexp.MustCompile(buffer.String())
	framePattern.Verbose = verbose

	return &framePattern
}

/**
* Parses all frame information from a given line.
* @param  line a line that represents a stack frame.
* @return the parsed information, or null if the line doesn't match a
*         stack frame.
 */
func (f *FramePattern) Parse(line string) FrameInfo {
	results := f.Pattern.FindStringSubmatch(line)

	var className, sourceFile, javaType, fieldName, methodName, arguments string
	var lineNumber int
	for i, result := range results {
		if len(result) == 0 {
			continue
		}

		switch f.ExpressionTypes[i] {
		case "c":
			className = result
		case "C":
			className = strings.ReplaceAll(result, "/", ".")
		case "s":
			sourceFile = result
		case "l":
			var err error
			lineNumber, err = strconv.Atoi(result)
			if err != nil {
				lineNumber = -1
			}
		case "t":
			javaType = result
		case "f":
			fieldName = result
		case "m":
			methodName = result
		case "a":
			arguments = result
		}
	}

	return FrameInfo{
		ClassName:  className,
		SourceFile: sourceFile,
		LineNumber: lineNumber,
		Type:       javaType,
		FieldName:  fieldName,
		MethodName: methodName,
		Arguments:  arguments,
	}
}

/**
 * Formats the given frame information based on the given template line.
 * It is the reverse of {@link #parse(String)}, but optionally with
 * different frame information.
 * @param  line      a template line that represents a stack frame.
 * @param  frameInfo information about a stack frame.
 * @return the formatted line, or null if the line doesn't match a
 *         stack frame.
 */
func (f *FramePattern) Format(line string, frameInfo FrameInfo) string {
	var formattedBuffer strings.Builder
	results := f.Pattern.FindStringSubmatchIndex(line)
	lineIndex := 0
	// Ignore the first result, which is the entire match.
	for expressionTypeIndex := 1; expressionTypeIndex < f.ExpressionTypeCount; expressionTypeIndex++ {
		matcherIndex := expressionTypeIndex * 2
		if matcherIndex > len(results) {
			break
		}
		startIndex := results[matcherIndex]
		if startIndex < 0 {
			continue
		}
		endIndex := results[matcherIndex+1]
		formattedBuffer.WriteString(line[lineIndex:startIndex])
		switch f.ExpressionTypes[expressionTypeIndex] {
		case "c":
			formattedBuffer.WriteString(frameInfo.ClassName)
		case "C":
			formattedBuffer.WriteString(strings.ReplaceAll(frameInfo.ClassName, ".", "/"))
		case "s":
			formattedBuffer.WriteString(frameInfo.SourceFile)
		case "l":
			formattedBuffer.WriteString(strconv.Itoa(frameInfo.LineNumber))
		case "t":
			formattedBuffer.WriteString(frameInfo.Type)
		case "f":
			if f.Verbose {
				formattedBuffer.WriteString(frameInfo.Type)
				formattedBuffer.WriteString(" ")
			}
			formattedBuffer.WriteString(frameInfo.FieldName)
		case "m":
			if f.Verbose {
				formattedBuffer.WriteString(frameInfo.Type)
				formattedBuffer.WriteString(" ")
			}
			formattedBuffer.WriteString(frameInfo.MethodName)
			if f.Verbose {
				formattedBuffer.WriteString("(")
				formattedBuffer.WriteString(frameInfo.Arguments)
				formattedBuffer.WriteString(")")
			}
		case "a":
			formattedBuffer.WriteString(frameInfo.Arguments)
		}
		lineIndex = endIndex
	}

	formattedBuffer.WriteString(line[lineIndex:])
	return formattedBuffer.String()
}
