package ptproc

import (
	"context"
	"errors"
	"regexp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/exp/slog"
)

var _ Rule = (*rangeImportRule)(nil)

var DefaultRangeImportStartRegEx = regexp.MustCompile(`range:(?P<RangeName>[^\s]+)`)
var DefaultRangeImportEndRegEx = regexp.MustCompile(`range.end`)

type rangeImportRule struct {
	targetName  string
	startRegExp *regexp.Regexp
	endRegExp   *regexp.Regexp
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
				name := group[1]
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
