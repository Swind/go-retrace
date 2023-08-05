package retrace

type FrameInfo struct {
	ClassName  string
	SourceFile string
	LineNumber int
	Type       string
	FieldName  string
	MethodName string
	Arguments  string
}
