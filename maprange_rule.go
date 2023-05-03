package ptproc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"cuelang.org/go/cue/cuecontext"
	"go.opentelemetry.io/otel"
	"golang.org/x/exp/slog"
)

var _ Rule = (*maprangeRule)(nil)

var DefaultMaprangeStartRegEx = regexp.MustCompile(`maprange:(?P<Cue>[^\s]+)`)
var DefaultMaprangeEndRegEx = regexp.MustCompile(`maprange.end`)

type MaprangeRuleConfig struct {
	StartRegExp *regexp.Regexp
	EndRegExp   *regexp.Regexp
	EmbedRules  []Rule
}

func NewMaprangeRule(cfg *MaprangeRuleConfig) (Rule, error) {
	if cfg == nil {
		cfg = &MaprangeRuleConfig{}
	}

	return &maprangeRule{
		startRegExp: cfg.StartRegExp,
		endRegExp:   cfg.EndRegExp,
		embedRules:  cfg.EmbedRules,
	}, nil
}

type maprangeRule struct {
	startRegExp *regexp.Regexp
	endRegExp   *regexp.Regexp

	embedRules []Rule
}

type maprangeParams struct {
	File string `cue:"file"`
	Name string `cue:"name"`
}

func (rule *maprangeRule) Apply(ctx context.Context, opts *RuleOptions, ns []Node) (_ []Node, err error) {
	ctx, span := otel.Tracer("ptproc").Start(ctx, "maprangeRule.Apply")
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	startRegExp := rule.startRegExp
	if startRegExp == nil {
		startRegExp = DefaultMaprangeStartRegEx
	}
	endRegExp := rule.endRegExp
	if endRegExp == nil {
		endRegExp = DefaultMaprangeEndRegEx
	}

	slog.DebugCtx(ctx, "start maprange rule processing")

	newNodes := make([]Node, 0, len(ns))

	var inMaprangeRange bool
	for _, n := range ns {
		txt := n.Text()

		if !inMaprangeRange {
			group := startRegExp.FindStringSubmatch(txt)

			if len(group) != 2 {
				newNodes = append(newNodes, n)
				continue
			}

			params, err := rule.textToParams(ctx, group[1])
			if err != nil {
				return nil, err
			}

			filePath := params.File
			realFilePath := opts.FilePath(filePath)
			rangeName := params.Name
			slog.DebugCtx(ctx, "find maprange directive",
				slog.String("filePath", filePath),
				slog.String("realFilePath", realFilePath),
				slog.String("rangeName", rangeName),
			)

			inMaprangeRange = true
			newNodes = append(newNodes, n)

			r, err := opts.OpenFile(realFilePath)
			if err != nil {
				return nil, err
			}
			b, err := io.ReadAll(r)
			if err != nil {
				return nil, err
			}
			s := string(b)

			rangeImportRule, err := NewRangeImportRule(&RangeImportRuleConfig{
				Name: rangeName,
			})
			if err != nil {
				return nil, err
			}

			embedRules := append([]Rule{rangeImportRule}, rule.embedRules...)

			subProc, err := opts.Processor.WithRules(ctx, embedRules)
			if err != nil {
				return nil, err
			}

			s, err = subProc.ProcessFile(ctx, realFilePath)
			if err != nil {
				return nil, err
			}

			if !strings.HasSuffix(s, "\n") {
				s += "\n"
			}

			newNodes = append(newNodes, &node{
				text: s,
			})
		} else if endRegExp.MatchString(txt) {
			inMaprangeRange = false
			newNodes = append(newNodes, n)
		} else {
			continue
		}
	}

	if inMaprangeRange {
		return nil, errors.New("maprange end directive is not found")
	}

	return newNodes, nil
}

func (rule *maprangeRule) textToParams(ctx context.Context, s string) (*maprangeParams, error) {
	cuectx := cuecontext.New()

	cv := cuectx.CompileString(s)

	err := cv.Validate()
	if err != nil {
		slog.DebugCtx(ctx, "cue validate failed. evaluate to string", "err", err, "value", s)
		ss := strings.SplitN(s, ",", 2)
		if len(ss) != 2 {
			return nil, fmt.Errorf("unexpected maprange syntax: %s", s)
		}

		return &maprangeParams{
			File: ss[0],
			Name: ss[1],
		}, nil
	}

	v, err := cv.String()
	if err == nil {
		ss := strings.SplitN(v, ",", 2)
		if len(ss) != 2 {
			return nil, fmt.Errorf("unexpected maprange syntax: %s", s)
		}

		return &maprangeParams{
			File: ss[0],
			Name: ss[1],
		}, nil
	} else {
		slog.DebugCtx(ctx, "failed to convert cue value to string. continue processing", "err", err, "value", s)
	}

	params := &maprangeParams{}
	err = cv.Decode(params)
	if err != nil {
		return nil, err
	}

	return params, nil
}
