package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/knaka/mdpp"
	"github.com/mattn/go-isatty"

	flag "github.com/spf13/pflag"

	. "github.com/knaka/go-utils" //revive:disable-line dot-imports
)

func main() {
	Debugger()
	var outPath string
	flag.StringVarP(&outPath, "outfile", "o", "", "Output outFile")
	var shouldPrintHelp bool
	flag.BoolVarP(&shouldPrintHelp, "help", "h", false, "Show Help")
	var inPlace bool
	flag.BoolVarP(&inPlace, "in-place", "i", false, "Edit file(s) in place")
	flag.Parse()
	if shouldPrintHelp {
		flag.Usage()
		os.Exit(0)
	}
	if inPlace {
		if outPath != "" {
			_, _ = fmt.Fprintln(os.Stderr, "Do not specify \"outfile\" and \"in-place\" simultaneously")
			os.Exit(1)
		}
	} else {
		if outPath == "" {
			outPath = "-"
		}
	}
	args := flag.Args()
	if inPlace {
		for _, inPath := range args {
			var err error
			func() {
				var inFile *os.File
				inFile, err = os.Open(inPath)
				if err != nil {
					log.Fatal("Failed to open inFile outFile: ", inPath)
				}
				defer func() { _ = inFile.Close() }()
				var outFile *os.File
				outFile, err = os.CreateTemp("", "mdpp")
				if err != nil {
					return
				}
				defer func() {
					_ = outFile.Close()
					_ = os.Remove(outFile.Name())
				}()
				bufOut := bufio.NewWriter(outFile)
				absPath := ""
				if inPath != "" {
					if absPath, err = filepath.Abs(inPath); err != nil {
						log.Fatal("Error", err.Error())
					}
				}
				sourceMD, err := os.ReadFile(absPath)
				if err != nil {
					log.Fatal("Failed to read inFile: ", inPath)
				}
				err = mdpp.Process(sourceMD, bufOut)
				if err != nil {
					log.Fatal("Failed to preprocess: ", err.Error())
				}
				// _, changed, err = mdpp.PreprocessOld(bufOut, inFile, filepath.Dir(inPath), absPath)
				// if err != nil {
				// 	return
				// }
				err = bufOut.Flush()
				if err != nil {
					return
				}
				err = inFile.Close()
				if err != nil {
					return
				}
				err = outFile.Close()
				if err != nil {
					return
				}
				// Compare the original file with the output file
				outContent, err := os.ReadFile(outFile.Name())
				if err != nil {
					log.Fatal("Failed to read output outFile: ", outFile.Name())
				}
				if bytes.Equal(sourceMD, outContent) {
					return
				}
				err = os.Rename(outFile.Name(), inPath)
				if err != nil {
					return
				}
			}()
			if err != nil {
				log.Fatalln("Failed to preprocess: ", err.Error())
			}
		}
	} else { // Not in-place mode
		var outFile *os.File
		var output io.Writer
		if outPath == "-" {
			outFile = os.Stdout
		} else {
			var err error
			outFile, err = os.OpenFile(outPath, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				log.Fatal("Failed to open output outFile: ", outPath)
			}
			defer func() { _ = outFile.Close() }()
		}
		if isatty.IsTerminal(outFile.Fd()) {
			output = outFile
		} else {
			bufOut := bufio.NewWriter(outFile)
			defer func() {
				_ = bufOut.Flush()
			}()
			output = bufOut
		}
		if len(args) == 0 {
			args = append(args, "-")
		}
		for _, inPath := range args {
			func() {
				var inFile *os.File
				if inPath == "-" {
					inFile = os.Stdin
				} else {
					var err error
					inFile, err = os.Open(inPath)
					if err != nil {
						log.Fatal("Failed to open inFile outFile: ", inPath)
					}
					defer func() { _ = inFile.Close() }()
				}
				sourceMD, err := io.ReadAll(inFile)
				if err != nil {
					log.Fatal("Failed to read inFile: ", inPath)
				}
				err = mdpp.Process(sourceMD, output)
				if err != nil {
					log.Fatal("Failed to preprocess: ", err.Error())
				}
			}()
		}
	}
}
