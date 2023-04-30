package ptproc

import (
	"context"
	"io"
	"path/filepath"
)

type Rule interface {
	Apply(ctx context.Context, opts *RuleOptions, nodes []Node) ([]Node, error)
}

type RuleOptions struct {
	Processor  Processor
	OpenFile   func(filePath string) (io.Reader, error)
	TargetPath string
}

func (opts *RuleOptions) FilePath(externalFilePath string) string {
	dirPath := filepath.Dir(opts.TargetPath)
	return filepath.Join(dirPath, externalFilePath)
}
