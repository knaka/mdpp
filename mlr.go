package mdpp

import (
	"os"

	mlrentry "github.com/johnkerl/miller/v6/pkg/entrypoint"
)

func mlrPutMDTableInplace(filePath string, script string) {
	argsSave := os.Args
	os.Args = []string{
		"mlr",
		"--imarkdown",
		"--omarkdown",
		"-I",
		"put",
		"-e", script,
		filePath,
	}
	defer func() { os.Args = argsSave }()
	mlrentry.Main()
}
