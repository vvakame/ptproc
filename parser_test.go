package ptproc

import (
	"context"
	"os"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestParse(t *testing.T) {
	ctx := context.Background()

	proc, err := NewProcessor(&ProcessorConfig{})
	if err != nil {
		t.Fatal(err)
	}

	s, err := proc.ProcessFile(ctx, "_misc/testdata/mapfile/base/test.md")
	if err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile("_misc/testdata/mapfile/expected/test.md")
	if err != nil {
		t.Fatal(err)
	}

	if v := string(b); s != v {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(v, s, false)

		t.Error(dmp.DiffPrettyText(diffs))
	}
}
