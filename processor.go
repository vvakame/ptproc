package ptproc

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/exp/slog"
)

type Processor interface {
	Parse(ctx context.Context, filePath string, r io.Reader) ([]Node, error)
	ProcessFile(ctx context.Context, filePath string) (string, error)
	WithRules(ctx context.Context, rules []Rule) (Processor, error)
}

type ProcessorConfig struct {
	OpenFile func(filePath string) (io.Reader, error)
	Rules    []Rule
}

func NewProcessor(cfg *ProcessorConfig) (Processor, error) {
	if cfg == nil {
		cfg = &ProcessorConfig{}
	}

	proc := &processor{
		openFile: cfg.OpenFile,
		rules:    cfg.Rules,
	}

	if proc.openFile == nil {
		proc.openFile = func(filePath string) (io.Reader, error) {
			return os.OpenFile(filePath, os.O_RDWR, 0o644)
		}
	}
	if len(proc.rules) == 0 {
		proc.rules = []Rule{
			&mapfileRule{},
			&maprangeRule{},
		}
	}

	return proc, nil
}

var _ Processor = (*processor)(nil)

type processor struct {
	openFile func(filePath string) (io.Reader, error)
	rules    []Rule
}

func (proc *processor) close() *processor {
	newProc := &processor{
		openFile: proc.openFile,
		rules:    proc.rules,
	}
	return newProc
}

func (proc *processor) ProcessFile(ctx context.Context, filePath string) (string, error) {
	slog.DebugCtx(ctx, "process file", slog.String("filePath", filePath))

	ns, err := proc.parseFile(ctx, filePath)
	if err != nil {
		return "", err
	}

	ns, err = proc.applyRules(ctx, filePath, ns)
	if err != nil {
		return "", err
	}

	s, err := proc.formatNodes(ctx, ns)
	if err != nil {
		return "", err
	}

	return s, nil
}

func (proc *processor) parseFile(ctx context.Context, filePath string) (_ []Node, err error) {
	ctx, span := otel.Tracer("ptproc").Start(ctx, "processor.parseFile")
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	r, err := proc.openFile(filePath)
	if err != nil {
		return nil, err
	}

	rc, ok := r.(io.ReadCloser)
	if ok {
		defer func() {
			err := rc.Close()
			if err != nil {
				slog.ErrorCtx(ctx, "file close")
			}
		}()
	}

	return proc.Parse(ctx, filePath, r)
}

func (proc *processor) Parse(ctx context.Context, filePath string, r io.Reader) (_ []Node, err error) {
	ctx, span := otel.Tracer("ptproc").Start(ctx, "processor.parse")
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()
	span.SetAttributes(attribute.String("filePath", filePath))

	result := make([]Node, 0)

	rdr := bufio.NewReader(r)
	for {
		l, err := rdr.ReadString('\n')
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}

		result = append(result, &node{
			text: l,
		})
	}

	return result, nil
}

func (proc *processor) applyRules(ctx context.Context, baseFilePath string, ns []Node) (_ []Node, err error) {
	ctx, span := otel.Tracer("ptproc").Start(ctx, "processor.applyRules")
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	span.SetAttributes(attribute.String("baseFilePath", baseFilePath), attribute.Int("nodeLength", len(ns)))

	for _, rule := range proc.rules {
		opts := &RuleOptions{
			Processor:  proc,
			OpenFile:   proc.openFile,
			TargetPath: baseFilePath,
		}
		ns, err = rule.Apply(ctx, opts, ns)
		if err != nil {
			return nil, err
		}
	}

	return ns, nil
}

func (proc *processor) formatNodes(ctx context.Context, ns []Node) (_ string, err error) {
	ctx, span := otel.Tracer("ptproc").Start(ctx, "processor.formatNodes")
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	var buf bytes.Buffer
	for _, n := range ns {
		buf.WriteString(n.Text())
	}

	return buf.String(), nil
}

func (proc *processor) WithRules(ctx context.Context, rules []Rule) (Processor, error) {
	proc = proc.close()
	proc.rules = rules
	return proc, nil
}
