package ptproc

import (
	"context"
	"errors"
	"regexp"

	"cuelang.org/go/cue/cuecontext"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/exp/slog"
)

var _ Rule = (*rangeImportRule)(nil)

var DefaultRangeImportStartRegEx = regexp.MustCompile(`range:(?P<Cue>[^\s]+)`)
var DefaultRangeImportEndRegEx = regexp.MustCompile(`range.end`)

type RangeImportRuleConfig struct {
	Name        string
	StartRegExp *regexp.Regexp
	EndRegExp   *regexp.Regexp
}

func NewRangeImportRule(cfg *RangeImportRuleConfig) (Rule, error) {
	if cfg == nil {
		cfg = &RangeImportRuleConfig{}
	}

	return &rangeImportRule{
		targetName:  cfg.Name,
		startRegExp: cfg.StartRegExp,
		endRegExp:   cfg.EndRegExp,
	}, nil
}

type rangeImportRule struct {
	targetName  string
	startRegExp *regexp.Regexp
	endRegExp   *regexp.Regexp
}

type rangeImportParams struct {
	Name string `cue:"name"`
}

func (rule *rangeImportRule) Apply(ctx context.Context, opts *RuleOptions, ns []Node) (_ []Node, err error) {
	ctx, span := otel.Tracer("ptproc").Start(ctx, "rangeImportRule.Apply")
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	span.SetAttributes(attribute.String("targetName", rule.targetName))

	startRegExp := rule.startRegExp
	if startRegExp == nil {
		startRegExp = DefaultRangeImportStartRegEx
	}
	endRegExp := rule.endRegExp
	if endRegExp == nil {
		endRegExp = DefaultRangeImportEndRegEx
	}

	slog.DebugCtx(ctx, "start range import rule processing")

	newNodes := make([]Node, 0, len(ns))

	var inRangeImportRange bool
	for _, n := range ns {
		txt := n.Text()

		if !inRangeImportRange {
			group := startRegExp.FindStringSubmatch(txt)

			if len(group) == 2 {
				params, err := rule.textToParams(ctx, group[1])
				if err != nil {
					return nil, err
				}

				name := params.Name
				slog.DebugCtx(ctx, "find range directive", slog.String("name", name))

				if name != rule.targetName {
					continue
				}

				inRangeImportRange = true
			}
		} else if endRegExp.MatchString(txt) {
			inRangeImportRange = false
		} else {
			newNodes = append(newNodes, n)
		}
	}

	if inRangeImportRange {
		return nil, errors.New("range end directive is not found")
	}

	return newNodes, nil
}

func (rule *rangeImportRule) textToParams(ctx context.Context, s string) (*rangeImportParams, error) {
	cuectx := cuecontext.New()

	cv := cuectx.CompileString(s)

	err := cv.Validate()
	if err != nil {
		slog.DebugCtx(ctx, "cue validate failed. evaluate to string", "err", err, "value", s)
		return &rangeImportParams{Name: s}, nil
	}

	v, err := cv.String()
	if err == nil {
		return &rangeImportParams{Name: v}, nil
	} else {
		slog.DebugCtx(ctx, "failed to convert cue value to string. continue processing", "err", err, "value", s)
	}

	params := &rangeImportParams{}
	err = cv.Decode(params)
	if err != nil {
		return nil, err
	}

	return params, nil
}
