package mdpp

import (
	"os"

	mlrentry "github.com/johnkerl/miller/v6/pkg/entrypoint"
)

func mlrPutMDTableInplace(filePath string, script string) {
	argsSave := os.Args
	defer func() { os.Args = argsSave }()
	os.Args = []string{
		"mlr",
		"--imarkdown",
		"--omarkdown",
		"-I",
		"put",
		"-e", script,
		filePath,
	}
	mlrentry.Main()
}
