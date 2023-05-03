package ptproc

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/vvakame/ptproc/internal/testutils"
)

func Test_LoadConfig(t *testing.T) {
	t.Parallel()

	const testFileDir = "./_misc/config"

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

			testDir := filepath.Join(testFileDir, dir.Name())

			configFilePath := filepath.Join(testDir, "testcase/ptproc.yaml")

			cfg, err := LoadConfig(ctx, configFilePath)
			if err != nil {
				t.Fatal(err)
			}

			b, err := yaml.MarshalContext(ctx, cfg)
			if err != nil {
				t.Fatal(err)
			}

			testutils.CheckGoldenFile(t, b, filepath.Join(testDir, "expected/ptproc.yaml"))

			textFilePath := filepath.Join(testDir, "testcase/test.md")
			_, err = os.Stat(textFilePath)
			if os.IsNotExist(err) {
				return
			} else if err != nil {
				t.Fatal(err)
			}

			procCfg, err := cfg.ToProcessorConfig(ctx)
			if err != nil {
				t.Fatal(err)
			}
			proc, err := NewProcessor(procCfg)
			if err != nil {
				t.Fatal(err)
			}

			s, err := proc.ProcessFile(ctx, textFilePath)
			if err != nil {
				t.Fatal(err)
			}

			testutils.CheckGoldenFile(t, []byte(s), filepath.Join(testDir, "expected/test.md"))
		})
	}
}
