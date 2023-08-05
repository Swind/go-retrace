package retrace

import (
	"bufio"
	"errors"
	"io"
	"strconv"
	"strings"
)

type MappingReader struct {
	fileReader io.Reader
}

func IndexOf(s string, subStr string, position int) int {
	index := strings.Index(s[position:], subStr)
	if index < 0 {
		return index
	} else {
		return position + index
	}
}

func NewMappingReader(fileReader io.Reader) *MappingReader {
	reader := MappingReader{
		fileReader: fileReader,
	}

	return &reader
}

func (r *MappingReader) Pump(processor MappingProcessor) error {
	var className string = ""

	scanner := bufio.NewScanner(r.fileReader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}

		// Is it a comment line ?
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Is it a class mapping or a class member mapping
		if strings.HasSuffix(line, ":") {
			// Process the class mapping and remember the class's old name
			className = r.ProcessClassMapping(line, processor)
		} else if len(className) > 0 {
			// Process the class member mapping, in the context of the current old class name
			r.ProcessClassMemberMapping(className, line, processor)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (r *MappingReader) ProcessClassMapping(line string, processor MappingProcessor) string {
	// See if we can parse "____ -> ____:", containing the original
	// class name and the new class name
	arrowIndex := IndexOf(line, "->", 0)
	if arrowIndex < 0 {
		return ""
	}

	colonIndex := IndexOf(line, ":", arrowIndex+2)
	if colonIndex < 0 {
		return ""
	}

	// Extract the elements
	className := strings.TrimSpace(line[0:arrowIndex])
	newClassName := strings.TrimSpace(line[arrowIndex+2 : colonIndex])

	// Process this class name mapping
	interested := processor.ProcessClassMapping(className, newClassName)
	if interested {
		return className
	}

	return ""
}

/**
 * Parses the given line with a class member mapping and processes the
 * results with the given mapping processor.
 */
func (r *MappingReader) ProcessClassMemberMapping(className string, line string, processor MappingProcessor) error {
	// See if we can parse one of
	//     ___ ___ -> ___
	//     ___:___:___ ___(___) -> ___
	//     ___:___:___ ___(___):___ -> ___
	//     ___:___:___ ___(___):___:___ -> ___
	// containing the optional line numbers, the return type, the original
	// field/method name, optional arguments, the optional original line
	// numbers, and the new field/method name. The original field/method
	// name may contain an original class name "___.___".

	var (
		colonIndex1 = -1
		colonIndex2 = -1
		colonIndex3 = -1
		colonIndex4 = -1

		argumentIndex1 = -1
		argumentIndex2 = -1

		spaceIndex = -1
		arrowIndex = -1

		cursor = -1
	)

	colonIndex1 = IndexOf(line, ":", 0)
	if colonIndex1 >= 0 {
		colonIndex2 = IndexOf(line, ":", colonIndex1+1)
	}

	spaceIndex = IndexOf(line, " ", colonIndex2+2)
	cursor = spaceIndex

	argumentIndex1 = IndexOf(line, "(", spaceIndex+1)
	if argumentIndex1 >= 0 {
		argumentIndex2 = IndexOf(line, ")", argumentIndex1+1)
	}
	if argumentIndex2 >= 0 {
		cursor = argumentIndex2
		colonIndex3 = IndexOf(line, ":", argumentIndex2+1)
	}
	if colonIndex3 >= 0 {
		cursor = colonIndex3
		colonIndex4 = IndexOf(line, ":", colonIndex3+1)
	}
	if colonIndex4 >= 0 {
		cursor = colonIndex4
	}

	arrowIndex = IndexOf(line, "->", cursor+1)

	if spaceIndex < 0 || arrowIndex < 0 {
		return errors.New("spaceIndex < 0 or arrowIndex < 0")
	}

	// Extract the elements
	classMemberType := strings.TrimSpace(line[colonIndex2+1 : spaceIndex])
	nameEndIndex := arrowIndex
	if argumentIndex1 >= 0 {
		nameEndIndex = argumentIndex1
	}
	classMemberName := strings.TrimSpace(line[spaceIndex+1 : nameEndIndex])
	newClassMemberName := strings.TrimSpace(line[arrowIndex+2:])

	// Does the method name contain an explicit original class name ?
	newClassName := className
	dotIndex := strings.LastIndex(classMemberName, ".")
	if dotIndex >= 0 {
		className = classMemberName[:dotIndex]
		classMemberName = classMemberName[dotIndex+1:]
	}

	// Process this class member mapping
	if len(classMemberType) > 0 && len(classMemberName) > 0 && len(newClassMemberName) > 0 {
		// Is it a field or a method
		if argumentIndex2 < 0 {
			processor.ProcessFieldMapping(className, classMemberType, classMemberName, newClassName, newClassMemberName)
		} else {
			var (
				firstLineNumber    = 0
				lastLineNumber     = 0
				newFirstLineNumber = 0
				newLastLineNumber  = 0
				err                error
			)

			if colonIndex2 >= 0 {
				firstLineNumber, err = strconv.Atoi(strings.TrimSpace(line[:colonIndex1]))
				if err != nil {
					return err
				}
				newFirstLineNumber = firstLineNumber

				lastLineNumber, err = strconv.Atoi(strings.TrimSpace(line[colonIndex1+1 : colonIndex2]))
				newLastLineNumber = lastLineNumber
			}

			if colonIndex3 >= 0 {
				firstLineNumberLastIndex := arrowIndex
				if colonIndex4 > 0 {
					firstLineNumberLastIndex = colonIndex4
				}
				firstLineNumber, err = strconv.Atoi(strings.TrimSpace(line[colonIndex3+1 : firstLineNumberLastIndex]))
				if err != nil {
					return err
				}

				if colonIndex4 < 0 {
					lastLineNumber = firstLineNumber
				} else {
					lastLineNumber, err = strconv.Atoi(strings.TrimSpace(line[colonIndex4+1 : arrowIndex]))
					if err != nil {
						return err
					}
				}
			}

			arguments := strings.TrimSpace(line[argumentIndex1+1 : argumentIndex2])
			processor.ProcessMethodMapping(
				className,
				firstLineNumber,
				lastLineNumber,
				classMemberType,
				classMemberName,
				arguments,
				newClassName,
				newFirstLineNumber,
				newLastLineNumber,
				newClassMemberName,
			)
		}
	}

	return nil
}
