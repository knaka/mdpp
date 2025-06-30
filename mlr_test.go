package mdpp

import (
	"bytes"
	"os"
	"path"
	"testing"

	"github.com/andreyvit/diff"

	. "github.com/knaka/go-utils"
)

func TestMLR(t *testing.T) {
	tempDirPath, tempDirCleanFn := mkdirTemp()
	defer tempDirCleanFn()
	original := []byte(`| Item | UnitPrice | Quantity | Total |
| --- | --- | --- | --- |
| Apple | 1.5 | 12 | 0 |
| Banana | 2.0 | 5 | 0 |
| Orange | 1.2 | 8 | 0 |
`)
	script := "$Total = $UnitPrice * $Quantity"
	expected := []byte(`| Item | UnitPrice | Quantity | Total |
| --- | --- | --- | --- |
| Apple | 1.5 | 12 | 18 |
| Banana | 2.0 | 5 | 10 |
| Orange | 1.2 | 8 | 9.6 |
`)
	tempFilePath := path.Join(tempDirPath, "data.csv")
	V0(os.WriteFile(tempFilePath, original, 0644))
	mlrMDInplacePut(tempFilePath, script)
	result := V(os.ReadFile(tempFilePath))
	if !bytes.Equal(expected, result) {
		t.Fatalf("MLR test failed:\n%s", diff.LineDiff(string(expected), string(result)))
	}
}
