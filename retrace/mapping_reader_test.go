package retrace

import (
	"compress/gzip"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockMappingProcessor struct{}

func (processor *MockMappingProcessor) ProcessClassMapping(className string, newClassName string) bool {
	return true
}
func (processor *MockMappingProcessor) ProcessFieldMapping(
	className string,
	fieldType string,
	fieldName string,
	newClassName string,
	newFieldName string) {
}
func (processor *MockMappingProcessor) ProcessMethodMapping(
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
}

func TestJavaExceptionParser(t *testing.T) {
	ioReader := strings.NewReader(`# compiler: R8
# compiler_version: 1.4.94
# min_api: 21
J.N -> J.N:
    1:1:void <init>():11:11 -> <init>
android.arch.core.executor.ArchTaskExecutor -> c:
    android.arch.core.executor.TaskExecutor mDelegate -> qb
    android.arch.core.executor.TaskExecutor mDefaultTaskExecutor -> ne
    1:1:void <clinit>():42:42 -> <clinit>
    2:2:void <clinit>():50:50 -> <clinit>
    1:3:void <init>():57:59 -> <init>
    1:1:void executeOnDiskIO(java.lang.Runnable):96:96 -> b
    1:1:void postToMainThread(java.lang.Runnable):101:101 -> c
    1:1:boolean isMainThread():116:116 -> ee
    1:2:android.arch.core.executor.ArchTaskExecutor getInstance():69:70 -> getInstance
    3:5:android.arch.core.executor.ArchTaskExecutor getInstance():72:74 -> getInstance
    6:7:android.arch.core.executor.ArchTaskExecutor getInstance():76:77 -> getInstance
    8:8:android.arch.core.executor.ArchTaskExecutor getInstance():76:76 -> getInstance
android.arch.core.executor.ArchTaskExecutor$1 -> a:
    1:1:void <init>():42:42 -> <init>
    1:1:void execute(java.lang.Runnable):45:45 -> execute
    2:2:void android.arch.core.executor.ArchTaskExecutor.postToMainThread(java.lang.Runnable):101:101 -> execute
    2:2:void execute(java.lang.Runnable):45 -> execute
android.arch.core.executor.ArchTaskExecutor$2 -> b:
    1:1:void <init>():50:50 -> <init>
    1:1:void execute(java.lang.Runnable):53:53 -> execute
    2:2:void android.arch.core.executor.ArchTaskExecutor.executeOnDiskIO(java.lang.Runnable):96:96 -> execute
    2:2:void execute(java.lang.Runnable):53 -> execute
android.arch.core.executor.DefaultTaskExecutor -> d:
    android.os.Handler mMainHandler -> pesfd
    java.util.concurrent.ExecutorService mDiskIO -> oe
    1:3:void <init>():31:33 -> <init>
    1:1:void executeOnDiskIO(java.lang.Runnable):40:40 -> b
    1:4:void postToMainThread(java.lang.Runnable):45:48 -> c
    5:5:void postToMainThread(java.lang.Runnable):50:50 -> c
    6:6:void postToMainThread(java.lang.Runnable):53:53 -> c
    1:1:boolean isMainThread():58:58 -> ee
android.arch.core.executor.TaskExecutor -> e:
    1:1:void <init>():31:31 -> <init>
    void executeOnDiskIO(java.lang.Runnable) -> b
    void postToMainThread(java.lang.Runnable) -> c
    boolean isMainThread() -> ee
android.arch.core.internal.FastSafeIterableMap -> f:
    java.util.HashMap mHashMap -> ze
    1:1:void <init>():35:35 -> <init>
    2:2:void <init>():37:37 -> <init>
    1:1:boolean contains(java.lang.Object):66:66 -> contains
    1:1:android.arch.core.internal.SafeIterableMap$Entry get(java.lang.Object):41:41 -> get
    1:1:android.arch.core.internal.SafeIterableMap$Entry get(java.lang.Object):41:41 -> putIfAbsent
    1:1:java.lang.Object putIfAbsent(java.lang.Object,java.lang.Object):46 -> putIfAbsent
    2:2:java.lang.Object putIfAbsent(java.lang.Object,java.lang.Object):48:48 -> putIfAbsent
    3:3:java.lang.Object putIfAbsent(java.lang.Object,java.lang.Object):50:50 -> putIfAbsent
    1:1:java.lang.Object android.arch.core.internal.SafeIterableMap.remove(java.lang.Object):97:97 -> remove
    1:1:java.lang.Object remove(java.lang.Object):56 -> remove
    2:5:java.lang.Object android.arch.core.internal.SafeIterableMap.remove(java.lang.Object):101:104 -> remove
    2:5:java.lang.Object remove(java.lang.Object):56 -> remove
    6:7:java.lang.Object android.arch.core.internal.SafeIterableMap.remove(java.lang.Object):108:109 -> remove
    6:7:java.lang.Object remove(java.lang.Object):56 -> remove
    8:8:java.lang.Object android.arch.core.internal.SafeIterableMap.remove(java.lang.Object):111:111 -> remove
    8:8:java.lang.Object remove(java.lang.Object):56 -> remove
    9:10:java.lang.Object android.arch.core.internal.SafeIterableMap.remove(java.lang.Object):114:115 -> remove
    9:10:java.lang.Object remove(java.lang.Object):56 -> remove
    11:11:java.lang.Object android.arch.core.internal.SafeIterableMap.remove(java.lang.Object):117:117 -> remove
    11:11:java.lang.Object remove(java.lang.Object):56 -> remove
    12:14:java.lang.Object android.arch.core.internal.SafeIterableMap.remove(java.lang.Object):120:122 -> remove
    12:14:java.lang.Object remove(java.lang.Object):56 -> remove
    15:15:java.lang.Object remove(java.lang.Object):57:57 -> remove
 `)

	mockProcessor := MockMappingProcessor{}
	mappingReader := NewMappingReader(ioReader)
	mappingReader.Pump(&mockProcessor)
}

func TestJavaExceptionParserForRetrace2(t *testing.T) {
	reader, err := os.Open("test_data/javaexception/symbol/mapping.txt.gz")
	assert.NoError(t, err)

	mappingFile, err := gzip.NewReader(reader)
	mappingReader := NewMappingReader(mappingFile)

	mockProcessor := MockMappingProcessor{}
	mappingReader.Pump(&mockProcessor)

}

func TestJavaExceptionParserForRetrace3(t *testing.T) {
	reader, err := os.Open("test_data/javaexception/symbol/mapping.3.txt.gz")
	mappingFile, err := gzip.NewReader(reader)
	mappingReader := NewMappingReader(mappingFile)

	mockProcessor := MockMappingProcessor{}
	mappingReader.Pump(&mockProcessor)

	assert.NoError(t, err)

}
