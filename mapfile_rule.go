package ptproc

import (
	"context"
	"cuelang.org/go/cue/cuecontext"
	"errors"
	"io"
	"regexp"
	"strings"

	"go.opentelemetry.io/otel"
	"golang.org/x/exp/slog"
)

var _ Rule = (*mapfileRule)(nil)

var DefaultMapfileStartRegEx = regexp.MustCompile(`mapfile:(?P<Cue>[^\s]+)`)
var DefaultMapfileEndRegEx = regexp.MustCompile(`mapfile.end`)

type MapfileRuleConfig struct {
	StartRegExp *regexp.Regexp
	EndRegExp   *regexp.Regexp
	DefaultSkip int
	EmbedRules  []Rule
}

func NewMapfileRule(cfg *MapfileRuleConfig) (Rule, error) {
	if cfg == nil {
		cfg = &MapfileRuleConfig{}
	}

	return &mapfileRule{
		startRegExp: cfg.StartRegExp,
		endRegExp:   cfg.EndRegExp,
		defaultSkip: cfg.DefaultSkip,
		embedRules:  cfg.EmbedRules,
	}, nil
}

type mapfileRule struct {
	startRegExp *regexp.Regexp
	endRegExp   *regexp.Regexp
	defaultSkip int

	embedRules []Rule
}

type mapfileParams struct {
	File string `cue:"file"`
	Skip *int   `cue:"skip"`
}

func (rule *mapfileRule) Apply(ctx context.Context, opts *RuleOptions, ns []Node) (_ []Node, err error) {
	ctx, span := otel.Tracer("ptproc").Start(ctx, "mapfileRule.Apply")
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	startRegExp := rule.startRegExp
	if startRegExp == nil {
		startRegExp = DefaultMapfileStartRegEx
	}
	endRegExp := rule.endRegExp
	if endRegExp == nil {
		endRegExp = DefaultMapfileEndRegEx
	}

	slog.DebugCtx(ctx, "start mapfile rule processing")

	newNodes := make([]Node, 0, len(ns))

	var inMapfileRange bool
	var realFilePath string
	var skip int
	var skipped int
	var skipBuffer []Node
	for _, n := range ns {
		txt := n.Text()

		if !inMapfileRange {
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
			realFilePath = opts.FilePath(filePath)
			skip = rule.defaultSkip
			if params.Skip != nil {
				skip = *params.Skip
			}
			skipped = 0
			slog.DebugCtx(ctx, "find mapfile directive",
				slog.String("filePath", filePath),
				slog.String("realFilePath", realFilePath),
				slog.Int("skip", skip),
			)

			inMapfileRange = true
			newNodes = append(newNodes, n)
		} else if endRegExp.MatchString(txt) {
			inMapfileRange = false
			head := len(skipBuffer) - skip
			if head < 0 {
				head = 0
			}

			s, err := rule.loadEmbed(ctx, opts, realFilePath)
			if err != nil {
				return nil, err
			}

			newNodes = append(newNodes, &node{
				text: s,
			})

			newNodes = append(newNodes, skipBuffer[head:]...)
			newNodes = append(newNodes, n)
			skipBuffer = nil
		} else if skipped < skip {
			newNodes = append(newNodes, n)
			skipped++
		} else {
			skipBuffer = append(skipBuffer, n)
		}
	}

	if inMapfileRange {
		return nil, errors.New("mapfile end directive is not found")
	}

	return newNodes, nil
}

func (rule *mapfileRule) loadEmbed(ctx context.Context, opts *RuleOptions, filePath string) (_ string, err error) {
	r, err := opts.OpenFile(filePath)
	if err != nil {
		return "", err
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	s := string(b)

	if len(rule.embedRules) != 0 {
		subProc, err := opts.Processor.WithRules(ctx, rule.embedRules)
		if err != nil {
			return "", err
		}

		s, err = subProc.ProcessFile(ctx, filePath)
		if err != nil {
			return "", err
		}
	}

	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}

	return s, nil
}

func (rule *mapfileRule) textToParams(ctx context.Context, s string) (*mapfileParams, error) {
	cuectx := cuecontext.New()

	cv := cuectx.CompileString(s)

	err := cv.Validate()
	if err != nil {
		slog.DebugCtx(ctx, "cue validate failed. evaluate to string", "err", err, "value", s)
		return &mapfileParams{File: s}, nil
	}

	v, err := cv.String()
	if err == nil {
		return &mapfileParams{File: v}, nil
	} else {
		slog.DebugCtx(ctx, "failed to convert cue value to string. continue processing", "err", err, "value", s)
	}

	params := &mapfileParams{}
	err = cv.Decode(params)
	if err != nil {
		return nil, err
	}

	return params, nil
}
