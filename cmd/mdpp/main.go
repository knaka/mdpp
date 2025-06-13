// Main for mdpp, a Markdown preprocessor
package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	flag "github.com/spf13/pflag"

	"github.com/knaka/mdpp"

	//revive:disable-next-line dot-imports
	. "github.com/knaka/go-utils"
)

const appID = "mdpp"

// stdinFileName is a special name for standard input.
const stdinFileName = "-"

func showUsage(cmdln *flag.FlagSet) {
	fmt.Fprintf(os.Stderr, "Usage: %s [options] [file...]\n", appID)
	cmdln.SetOutput(os.Stderr)
	cmdln.PrintDefaults()
}

func mdppMain(args []string) (err error) {
	cmdln := flag.NewFlagSet(appID, flag.ContinueOnError)

	var shouldPrintHelp bool
	cmdln.BoolVarP(&shouldPrintHelp, "help", "h", false, "Show help")

	var inPlace bool
	cmdln.BoolVarP(&inPlace, "in-place", "i", false, "Edit file(s) in-place")

	var debugMode bool
	cmdln.BoolVarP(&debugMode, "debug", "d", false, "Enable debug mode")

	err = cmdln.Parse(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		showUsage(cmdln)
		return errors.New("")
	}

	if shouldPrintHelp {
		showUsage(cmdln)
		return nil
	}
	if debugMode {
		mdpp.SetDebug(true)
	}
	args = cmdln.Args()
	if len(args) == 0 {
		args = append(args, stdinFileName)
	}
	for _, inPath := range args {
		err = func() error {
			var inDirPath string
			var inFile *os.File
			if inPath == stdinFileName {
				inDirPath = "."
				if inPlace {
					return fmt.Errorf("Cannot use in-place mode with standard input")
				}
				inFile = os.Stdin
			} else {
				inPath, err = filepath.EvalSymlinks(inPath)
				if err != nil {
					return fmt.Errorf("Failed to evaluate symlinks for inPath: %s Error: %v", inPath, err)
				}
				inDirPath = filepath.Dir(inPath)
				inFile, err = os.Open(inPath)
				if err != nil {
					return fmt.Errorf("Failed to open inFile outFile: %s Error: %v", inPath, err)
				}
				defer inFile.Close()
			}
			var outFile *os.File
			if inPlace {
				outFile, err = os.CreateTemp("", appID)
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
			var sourceMD []byte
			sourceMD, err = io.ReadAll(inFile)
			if err != nil {
				return fmt.Errorf("Failed to read inFile: %s Error: %v", inPath, err)
			}
			bufOut := bufio.NewWriter(outFile)
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
					return fmt.Errorf("Failed to close outFile: %s Error: %v", outFile.Name(), err)
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
				// Replace the original file with the output file
				err = os.Rename(outFile.Name(), inPath)
				if err != nil {
					return fmt.Errorf("Failed to rename temporary outFile to inPath: %s Error: %v", inPath, err)
				}
			}
			return nil
		}()
		if err != nil {
			break
		}
	}
	return err
}

func main() {
	Debugger()
	err := mdppMain(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
}
