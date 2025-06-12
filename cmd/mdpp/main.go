// Main for mdpp, a Markdown preprocessor
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	flag "github.com/spf13/pflag"

	"github.com/knaka/mdpp"

	. "github.com/knaka/go-utils" //revive:disable-line dot-imports
)

const stdinFileName = "-"

func mdppMain(args []string) error {
	var err error

	Debugger()

	commandLine := flag.NewFlagSet("mdpp", flag.ContinueOnError)

	var shouldPrintHelp bool
	commandLine.BoolVarP(&shouldPrintHelp, "help", "h", false, "Show Help")

	var inPlace bool
	commandLine.BoolVarP(&inPlace, "in-place", "i", false, "Edit file(s) in place")

	err = commandLine.Parse(args)
	if err != nil {
		return fmt.Errorf("%s\nUsage of mdpp:\n%s", err.Error(), commandLine.FlagUsages())
	}

	if shouldPrintHelp {
		fmt.Fprintf(os.Stdout, "Usage of mdpp:\n")
		commandLine.PrintDefaults()
		return nil
	}
	args = commandLine.Args()
	if len(args) == 0 {
		args = append(args, stdinFileName)
	}
	for _, inPath := range args {
		inPath, err = filepath.EvalSymlinks(inPath)
		if err != nil {
			return fmt.Errorf("Failed to evaluate symlinks for inPath: %s Error: %v", inPath, err)
		}
		inDirPath := filepath.Dir(inPath)
		err = func() error {
			var inFile *os.File
			if inPath == stdinFileName {
				if inPlace {
					return fmt.Errorf("Cannot use in-place mode with standard input")
				}
				inFile = os.Stdin
			} else {
				inFile, err = os.Open(inPath)
				if err != nil {
					return fmt.Errorf("Failed to open inFile outFile: %s Error: %v", inPath, err)
				}
				defer inFile.Close()
			}
			var outFile *os.File
			if inPlace {
				outFile, err = os.CreateTemp("", "mdpp")
				if err != nil {
					return fmt.Errorf("Failed to create temporary outFile: %v", err)
				}
				defer func() {
					_ = outFile.Close()
					_ = os.Remove(outFile.Name())
				}()
			} else {
				outFile = os.Stdout
			}
			bufOut := bufio.NewWriter(outFile)
			var sourceMD []byte
			sourceMD, err = os.ReadFile(inPath)
			if err != nil {
				return fmt.Errorf("Failed to read inFile: %s Error: %v", inPath, err)
			}
			err = mdpp.Process(sourceMD, bufOut, inDirPath)
			if err != nil {
				return fmt.Errorf("Failed to preprocess: %v", err)
			}
			err = bufOut.Flush()
			if err != nil {
				return fmt.Errorf("Failed to flush output: %v", err)
			}
			if inFile != os.Stdin {
				err = inFile.Close()
				if err != nil {
					return fmt.Errorf("Failed to close inFile: %s Error: %v", inPath, err)
				}
			}
			if outFile != os.Stdout {
				err = outFile.Close()
				if err != nil {
					return nil
				}
			}
			if inPlace {
				// Compare the original file with the output file
				var outContent []byte
				outContent, err = os.ReadFile(outFile.Name())
				if err != nil {
					return fmt.Errorf("Failed to read output outFile: %s", outFile.Name())
				}
				if bytes.Equal(sourceMD, outContent) {
					return nil
				}
				err = os.Rename(outFile.Name(), inPath)
				if err != nil {
					return fmt.Errorf("Failed to rename temporary outFile to inPath: %s Error: %v", inPath, err)
				}
			}
			return nil
		}()
		if err != nil {
			return fmt.Errorf("Failed to preprocess: %v", err)
		}
	}
	return nil
}

func main() {
	err := mdppMain(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
}
