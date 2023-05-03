package ptproc

import (
	"context"
	"errors"
	"io"
	"regexp"
	"strings"

	"go.opentelemetry.io/otel"
	"golang.org/x/exp/slog"
)

var _ Rule = (*mapfileRule)(nil)

var DefaultMapfileStartRegEx = regexp.MustCompile(`mapfile:(?P<FilePath>[^\s]+)`)
var DefaultMapfileEndRegEx = regexp.MustCompile(`mapfile.end`)

type MapfileRuleConfig struct {
	StartRegExp *regexp.Regexp
	EndRegExp   *regexp.Regexp
	EmbedRules  []Rule
}

func NewMapfileRule(cfg *MapfileRuleConfig) (Rule, error) {
	if cfg == nil {
		cfg = &MapfileRuleConfig{}
	}

	return &mapfileRule{
		startRegExp: cfg.StartRegExp,
		endRegExp:   cfg.EndRegExp,
		embedRules:  cfg.EmbedRules,
	}, nil
}

type mapfileRule struct {
	startRegExp *regexp.Regexp
	endRegExp   *regexp.Regexp

	embedRules []Rule
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
	for _, n := range ns {
		txt := n.Text()

		if !inMapfileRange {
			group := startRegExp.FindStringSubmatch(txt)

			var filePath string
			for idx, name := range startRegExp.SubexpNames() {
				switch name {
				case "FilePath":
					if idx < len(group) {
						filePath = group[idx]
					}
				}
			}

			if filePath != "" {
				realFilePath := opts.FilePath(filePath)
				slog.DebugCtx(ctx, "find mapfile directive",
					slog.String("filePath", filePath),
					slog.String("realFilePath", realFilePath),
				)

				inMapfileRange = true
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

				if len(rule.embedRules) != 0 {
					subProc, err := opts.Processor.WithRules(ctx, rule.embedRules)
					if err != nil {
						return nil, err
					}

					s, err = subProc.ProcessFile(ctx, realFilePath)
					if err != nil {
						return nil, err
					}
				}

				if !strings.HasSuffix(s, "\n") {
					s += "\n"
				}

				newNodes = append(newNodes, &node{
					text: s,
				})
			} else {
				newNodes = append(newNodes, n)
			}
		} else if endRegExp.MatchString(txt) {
			inMapfileRange = false
			newNodes = append(newNodes, n)
		} else {
			continue
		}
	}

	if inMapfileRange {
		return nil, errors.New("mapfile end directive is not found")
	}

	return newNodes, nil
}
