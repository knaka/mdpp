package mdpp

import (
	"os"

	mlrentry "github.com/johnkerl/miller/v6/pkg/entrypoint"
)

// mlrMDInplacePut runs Miller with the specified file path and script for processing.
// filePath is the path to the input file. Miller, as a library, does not support processing data in memory.
func mlrMDInplacePut(filePath string, script string) {
	argsSave := os.Args
	defer func() { os.Args = argsSave }()
	os.Args = []string{
		"mlr",
		// List of command-line flags - Miller Documentation https://miller.readthedocs.io/en/latest/reference-main-flag-list/
		// File formats - Miller Documentation https://miller.readthedocs.io/en/latest/file-formats/
		"--imarkdown",
		"--omarkdown",
		// In-place mode - Miller Documentation https://miller.readthedocs.io/en/latest/reference-main-in-place-processing/
		"-I",
		// List of verbs - Miller Documentation https://miller.readthedocs.io/en/latest/reference-verbs/#put
		"put",
		"-e", script,
		filePath,
	}
	mlrentry.Main()
}
