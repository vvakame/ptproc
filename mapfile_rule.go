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

type mapfileRule struct {
	startRegExp *regexp.Regexp
	endRegExp   *regexp.Regexp
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
			if len(group) == 2 {
				filePath := group[1]
				slog.DebugCtx(ctx, "find mapfile directive", slog.String("filePath", filePath))

				inMapfileRange = true
				newNodes = append(newNodes, n)

				r, err := opts.OpenFile(filePath)
				if err != nil {
					return nil, err
				}
				b, err := io.ReadAll(r)
				if err != nil {
					return nil, err
				}
				s := string(b)
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
