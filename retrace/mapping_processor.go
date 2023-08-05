package retrace

import (
	"strings"

	"github.com/emirpasic/gods/sets/hashset"
	"github.com/emirpasic/gods/sets/linkedhashset"
)

// Reference: https://github.com/Guardsquare/proguard/blob/0344c58b3d43799ce203737eea3fd1b58ca701ad/retrace/src/proguard/retrace/FrameRemapper.java
// MappingProcessor This interface specifies methods to process name mappings between original
// classes and their obfuscated versions. The mappings are typically read
// from a mapping file.
type MappingProcessor interface {
	// ProcessClassMapping Processes the given class name mapping.
	//
	// Parameters:
	//    className    the original class name.
	//    newClassName the new class name.
	//
	// Return:
	//    whether the processor is interested in receiving mappings of the
	//    class members of this class.
	ProcessClassMapping(className string, newClassName string) bool

	// ProcessFieldMapping
	// Processes the given field name mapping.
	//
	// Parameters:
	//    className    the original class name.
	//    fieldType    the original external field type.
	//    fieldName    the original field name.
	//    newClassName the new class name.
	//    newFieldName the new field name.
	ProcessFieldMapping(
		className string,
		fieldType string,
		fieldName string,
		newClassName string,
		newFieldName string)

	// ProcessMethodMapping
	// Processes the given method name mapping.
	// Parameters:
	//    className          the original class name.
	//    firstLineNumber    the first line number of the method, or 0 if
	//                       it is not known.
	//    lastLineNumber     the last line number of the method, or 0 if
	//                       it is not known.
	//    methodReturnType   the original external method return type.
	//    methodName         the original external method name.
	//    methodArguments    the original external method arguments.
	//    newClassName       the new class name.
	//    newFirstLineNumber the new first line number of the method, or 0
	//                       if it is not known.
	//    newLastLineNumber  the new last line number of the method, or 0
	//                       if it is not known.
	//    newMethodName      the new method name.
	ProcessMethodMapping(
		className string,
		firstLineNumber int,
		lastLineNumber int,
		methodType string,
		methodName string,
		arguments string,
		newClassName string,
		newFirstLineNumber int,
		newLastLineNumber int,
		newMethodName string)
}

type FieldInfo struct {
	OriginalClassName string
	OriginalType      string
	OriginalName      string
}

// Matches return whether the given type matches the original type of this field.
// The given type may be a null wildcard.
func (info *FieldInfo) Matches(originalType string) bool {
	return originalType == "" || originalType == info.OriginalType
}

type MethodInfo struct {
	ObfuscatedFirstLineNumber int
	ObfuscatedLastLineNumber  int

	OriginalClassName       string
	OriginalFirstLineNumber int
	OriginalLastLineNumber  int
	OriginalType            string
	OriginalName            string
	OriginalArguments       string
}

func (info *MethodInfo) Matches(obfuscatedLineNumber int, originalType string, originalArguments string) bool {
	return (obfuscatedLineNumber == 0 ||
		info.ObfuscatedLastLineNumber == 0 ||
		(info.ObfuscatedFirstLineNumber <= obfuscatedLineNumber && obfuscatedLineNumber <= info.ObfuscatedLastLineNumber)) &&
		(originalType == "" || originalType == info.OriginalType) &&
		(originalArguments == "" || originalArguments == info.OriginalArguments)

}

type ObfuscatedNameFieldInfoSetMap map[string]*hashset.Set
type ObfuscatedNameMethodInfoSetMap map[string]*linkedhashset.Set

type FrameRemapper struct {
	// ClassMap Obfuscated class name -> original class name.
	ClassMap map[string]string
	// ClassFieldMap Original class name -> obfuscated member name -> member info set.
	ClassFieldMap  map[string]ObfuscatedNameFieldInfoSetMap
	ClassMethodMap map[string]ObfuscatedNameMethodInfoSetMap
}

func NewFrameRemapper() *FrameRemapper {
	remapper := FrameRemapper{
		ClassMap:       make(map[string]string),
		ClassFieldMap:  make(map[string]ObfuscatedNameFieldInfoSetMap),
		ClassMethodMap: make(map[string]ObfuscatedNameMethodInfoSetMap),
	}

	return &remapper
}

func (remapper *FrameRemapper) ProcessClassMapping(className string, newClassName string) bool {
	// Obfuscated class name -> original class name
	remapper.ClassMap[newClassName] = className
	return true
}

func (remapper *FrameRemapper) ProcessFieldMapping(
	className string,
	fieldType string,
	fieldName string,
	newClassName string,
	newFieldName string) {

	// Obfuscated class name -> obfuscated field names
	var fieldMap ObfuscatedNameFieldInfoSetMap
	var ok bool
	fieldMap, ok = remapper.ClassFieldMap[newClassName]
	if !ok {
		fieldMap = make(ObfuscatedNameFieldInfoSetMap)
		remapper.ClassFieldMap[newClassName] = fieldMap
	}

	// Obfuscated field name -> fields
	var fieldInfoSet *hashset.Set
	fieldInfoSet, ok = fieldMap[newFieldName]
	if !ok {
		fieldInfoSet = hashset.New()
		fieldMap[newFieldName] = fieldInfoSet
	}

	// Add the field information
	fieldInfoSet.Add(FieldInfo{
		OriginalClassName: className,
		OriginalType:      fieldType,
		OriginalName:      fieldName,
	})
}

func (remapper *FrameRemapper) ProcessMethodMapping(
	className string,
	firstLineNumber int,
	lastLineNumber int,
	methodType string,
	methodName string,
	arguments string,
	newClassName string,
	newFirstLineNumber int,
	newLastLineNumber int,
	newMethodName string) {

	// Original class name -> obfuscated method names
	var methodMap ObfuscatedNameMethodInfoSetMap
	var ok bool
	methodMap, ok = remapper.ClassMethodMap[newClassName]
	if !ok {
		methodMap = make(ObfuscatedNameMethodInfoSetMap)
		remapper.ClassMethodMap[newClassName] = methodMap
	}

	// Obfuscated method name -> methods
	var methodInfoSet *linkedhashset.Set
	methodInfoSet, ok = methodMap[newMethodName]
	if !ok {
		methodInfoSet = linkedhashset.New()
		methodMap[newMethodName] = methodInfoSet
	}

	// Add the method information
	methodInfoSet.Add(MethodInfo{
		newFirstLineNumber,
		newLastLineNumber,
		className,
		firstLineNumber,
		lastLineNumber,
		methodType,
		methodName,
		arguments,
	})
}

func (remapper *FrameRemapper) Transform(obfuscatedFrame *FrameInfo) []FrameInfo {
	// First remap the class name.
	originalClassName := remapper.GetOriginalClassName(obfuscatedFrame.ClassName)

	// Create any transformed frames with remapped field names.
	var originalFrames []FrameInfo
	originalFrames = remapper.transformFieldInfo(*obfuscatedFrame, originalClassName, originalFrames)

	// Create any transformed frames with remapped method names.
	originalFrames = remapper.transformMethodInfo(*obfuscatedFrame, originalClassName, originalFrames)

	if len(originalFrames) == 0 {
		// No remapping was possible, so just use the original frame.
		var sourceFile string = obfuscatedFrame.SourceFile
		if len(sourceFile) == 0 && sourceFile != "Unknown Source" && sourceFile != "Native Method" {
			sourceFile = remapper.getSourceFileName(originalClassName)
		}

		originalFrames = append(originalFrames, FrameInfo{
			originalClassName,
			sourceFile,
			obfuscatedFrame.LineNumber,
			obfuscatedFrame.Type,
			obfuscatedFrame.FieldName,
			obfuscatedFrame.MethodName,
			obfuscatedFrame.Arguments,
		})
	}

	return originalFrames
}

/**
 * transformFieldInfo
 * Transforms the obfuscated frame into one or more original frames,
 * if the frame contains information about a field that can be remapped.
 * @param obfuscatedFrame     the obfuscated frame.
 * @param originalFieldFrames the list in which remapped frames can be
 *                            collected.
 */
func (remapper *FrameRemapper) transformFieldInfo(obfuscatedFrame FrameInfo, originalClassName string, originalFieldFrames []FrameInfo) []FrameInfo {
	// Class name -> obfuscated field names
	fieldMap, ok := remapper.ClassFieldMap[originalClassName]
	if !ok {
		return originalFieldFrames
	}

	// Obfuscated field names -> fields
	obfuscatedFieldName := obfuscatedFrame.FieldName
	fieldSet, ok := fieldMap[obfuscatedFieldName]
	if !ok {
		return originalFieldFrames
	}

	originalType := remapper.getOriginalType(obfuscatedFrame.Type)

	// Find all matching fields
	for _, item := range fieldSet.Values() {
		fieldInfo := item.(FieldInfo)
		if !fieldInfo.Matches(originalType) {
			continue
		}

		originalFieldFrames = append(originalFieldFrames, FrameInfo{
			fieldInfo.OriginalClassName,
			remapper.getSourceFileName(fieldInfo.OriginalClassName),
			obfuscatedFrame.LineNumber,
			fieldInfo.OriginalType,
			fieldInfo.OriginalName,
			obfuscatedFrame.MethodName,
			obfuscatedFrame.Arguments,
		})
	}

	return originalFieldFrames
}

/**
 * transformMethodInfo
 * Transforms the obfuscated frame into one or more original frames,
 * if the frame contains information about a method that can be remapped.
 * @param obfuscatedFrame      the obfuscated frame.
 * @param originalMethodFrames the list in which remapped frames can be
 *                             collected.
 */
func (remapper *FrameRemapper) transformMethodInfo(obfuscatedFrame FrameInfo, originalClassName string, originalMethodFrames []FrameInfo) []FrameInfo {
	// Class name -> obfuscated method names
	methodMap, ok := remapper.ClassMethodMap[originalClassName]
	if !ok {
		return originalMethodFrames
	}

	// Obfuscated method names -> methods
	obfuscatedMethodName := obfuscatedFrame.MethodName
	methodSet, ok := methodMap[obfuscatedMethodName]
	if !ok {
		return originalMethodFrames
	}

	obfuscatedLineNumber := obfuscatedFrame.LineNumber
	originalType := remapper.getOriginalType(obfuscatedFrame.Type)
	originalArguments := remapper.getOriginalArguments(obfuscatedFrame.Arguments)

	// Find all matching methods
	for _, item := range methodSet.Values() {
		methodInfo := item.(MethodInfo)
		if !methodInfo.Matches(obfuscatedLineNumber, originalType, originalArguments) {
			continue
		}

		lineNumber := obfuscatedFrame.LineNumber
		if methodInfo.OriginalFirstLineNumber != methodInfo.ObfuscatedFirstLineNumber {
			if methodInfo.OriginalLastLineNumber != 0 &&
				methodInfo.OriginalLastLineNumber != methodInfo.OriginalFirstLineNumber &&
				methodInfo.ObfuscatedFirstLineNumber != 0 &&
				lineNumber != 0 {
				lineNumber = methodInfo.OriginalFirstLineNumber - methodInfo.ObfuscatedFirstLineNumber + lineNumber
			} else {
				lineNumber = methodInfo.OriginalFirstLineNumber
			}
		}

		originalMethodFrames = append(originalMethodFrames, FrameInfo{
			methodInfo.OriginalClassName,
			remapper.getSourceFileName(methodInfo.OriginalClassName),
			lineNumber,
			methodInfo.OriginalType,
			obfuscatedFrame.FieldName,
			methodInfo.OriginalName,
			methodInfo.OriginalArguments,
		})
	}

	return originalMethodFrames
}

func (remapper *FrameRemapper) getSourceFileName(className string) string {
	if len(className) == 0 {
		return className
	}

	index1 := strings.LastIndex(className, ".") + 1
	index2 := IndexOf(className, "$", index1)

	if index2 > 0 {
		return className[index1:index2]
	} else {
		return className[index1:] + ".java"
	}
}

func (remapper *FrameRemapper) getOriginalType(obfuscatedType string) string {
	index := strings.Index(obfuscatedType, "[")
	if index >= 0 {
		return remapper.GetOriginalClassName(obfuscatedType[0:index]) + obfuscatedType[index:]
	} else {
		return remapper.GetOriginalClassName(obfuscatedType)
	}
}

func (remapper *FrameRemapper) GetOriginalClassName(obfuscatedClassName string) string {
	originalClassName, ok := remapper.ClassMap[obfuscatedClassName]
	if !ok {
		return obfuscatedClassName
	} else {
		return originalClassName
	}
}

func (remapper *FrameRemapper) getOriginalArguments(obfuscatedArguments string) string {
	tokens := strings.Split(obfuscatedArguments, ",")
	if len(tokens) < 1 {
		return ""
	}

	originalArguments := make([]string, len(tokens))

	for index, token := range tokens {
		originalArguments[index] = remapper.getOriginalType(strings.TrimSpace(token))
	}

	return strings.Join(originalArguments, ",")
}
