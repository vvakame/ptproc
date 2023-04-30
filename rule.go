package ptproc

import (
	"context"
	"io"
)

type Rule interface {
	Apply(ctx context.Context, opts *RuleOptions, nodes []Node) ([]Node, error)
}

type RuleOptions struct {
	OpenFile func(filePath string) (io.Reader, error)
}
