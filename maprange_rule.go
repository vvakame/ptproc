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

var _ Rule = (*maprangeRule)(nil)

var DefaultMaprangeStartRegEx = regexp.MustCompile(`maprange:(?P<FilePath>[^\s,]+),(?P<RangeName>[^\s]+)`)
var DefaultMaprangeEndRegEx = regexp.MustCompile(`maprange.end`)

type maprangeRule struct {
	startRegExp *regexp.Regexp
	endRegExp   *regexp.Regexp
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
			if len(group) == 3 {
				filePath := group[1]
				realFilePath := opts.FilePath(filePath)
				rangeName := group[2]
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

				subProc, err := opts.Processor.WithRules(ctx, []Rule{
					&rangeImportRule{
						targetName: rangeName,
					},
					&dedentRule{},
				})
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
			} else {
				newNodes = append(newNodes, n)
			}
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
