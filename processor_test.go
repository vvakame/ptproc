package ptproc

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vvakame/ptproc/internal/testutils"
)

func Test_process(t *testing.T) {
	t.Parallel()

	const testFileDir = "./_misc/testdata"

	dirs, err := os.ReadDir(testFileDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		dir := dir

		t.Run(dir.Name(), func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			filePath := filepath.Join(testFileDir, dir.Name(), "base/test.md")

			proc, err := NewProcessor(nil)
			if err != nil {
				t.Fatal(err)
			}

			s, err := proc.ProcessFile(ctx, filePath)
			if err != nil {
				t.Fatal(err)
			}

			testutils.CheckGoldenFile(t, []byte(s), filepath.Join(testFileDir, dir.Name(), "expected/test.md"))
		})
	}
}
