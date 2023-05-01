package ptproc

import (
	"context"
	"regexp"
	"strings"

	"go.opentelemetry.io/otel"
)

var _ Rule = (*dedentRule)(nil)

var DefaultDedentSpaceRegEx = regexp.MustCompile(`^(?P<Space>[\s]+)`)

type dedentRule struct {
	spaceRegExp *regexp.Regexp
}

func (rule *dedentRule) Apply(ctx context.Context, opts *RuleOptions, ns []Node) (_ []Node, err error) {
	ctx, span := otel.Tracer("ptproc").Start(ctx, "dedentRule.Apply")
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	if len(ns) == 0 {
		return nil, nil
	}

	spaceRegExp := rule.spaceRegExp
	if spaceRegExp == nil {
		spaceRegExp = DefaultDedentSpaceRegEx
	}

	newNodes := make([]Node, 0, len(ns))

	h := ns[0]
	txt := h.Text()
	group := spaceRegExp.FindStringSubmatch(txt)
	if len(group) != 2 {
		return ns, nil
	}
	space := group[1]

	newNodes = append(newNodes, &node{strings.TrimPrefix(txt, space)})

	for _, n := range ns[1:] {
		txt := n.Text()

		newNodes = append(newNodes, &node{strings.TrimPrefix(txt, space)})
	}

	return newNodes, nil
}
