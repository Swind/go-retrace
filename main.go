package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/swind/go-retrace/retrace"
)

func main() {
	args := os.Args[1:]

	if len(args) < 2 {
		fmt.Printf("Usage: %s <mapping file> <crash log file>\n", os.Args[0])
		os.Exit(1)
	}

	// The first argument is the mapping file
	mappingFilePath := args[0]

	// Check mappingFile exists
	if _, err := os.Stat(mappingFilePath); os.IsNotExist(err) {
		fmt.Printf("Mapping file %s does not exist\n", mappingFilePath)
		os.Exit(1)
	}

	// The second argument is the crash log file
	crashLogFilePath := args[1]
	// Check crashLogFile exists
	if _, err := os.Stat(crashLogFilePath); os.IsNotExist(err) {
		fmt.Printf("Crash log file %s does not exist\n", crashLogFilePath)
		os.Exit(1)
	}

	// Read the mapping file
	mappingFile, err := os.Open(mappingFilePath)
	if err != nil {
		fmt.Printf("Error opening mapping file: %s\n", err)
		os.Exit(1)
	}

	var mappingFileReader io.Reader
	if strings.HasSuffix(mappingFilePath, ".gz") {
		mappingFileReader, err = gzip.NewReader(mappingFile)
		if err != nil {
			fmt.Printf("Error opening mapping file: %s\n", err)
			os.Exit(1)
		}
	} else {
		mappingFileReader = bufio.NewReader(mappingFile)
	}

	retrace := retrace.NewRetrace(mappingFileReader)

	// Read the crash log file
	crashLogFile, err := os.Open(crashLogFilePath)
	if err != nil {
		fmt.Printf("Error opening crash log file: %s\n", err)
		os.Exit(1)
	}

	var crashLogFileReader io.Reader
	if strings.HasSuffix(crashLogFilePath, ".gz") {
		crashLogFileReader, err = gzip.NewReader(crashLogFile)
		if err != nil {
			fmt.Printf("Error opening crash log file: %s\n", err)
			os.Exit(1)
		}
	} else {
		crashLogFileReader = bufio.NewReader(crashLogFile)
	}

	resultBuffer := bytes.NewBufferString("")
	retrace.Retrace(crashLogFileReader, resultBuffer)

	fmt.Printf("%s", resultBuffer.String())
}
