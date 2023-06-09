package ptproc

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel"
)

var _ Rule = (*reindentRule)(nil)

const DefaultIndentLevel = 2

type ReindentRuleConfig struct {
	IndentLevel int
}

func NewReindentRule(cfg *ReindentRuleConfig) (Rule, error) {
	if cfg == nil {
		cfg = &ReindentRuleConfig{}
	}

	return &reindentRule{
		indentLevel: cfg.IndentLevel,
	}, nil
}

type reindentRule struct {
	indentLevel int
}

func (rule *reindentRule) Apply(ctx context.Context, opts *RuleOptions, ns []Node) (_ []Node, err error) {
	ctx, span := otel.Tracer("ptproc").Start(ctx, "reindentRule.Apply")
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	if len(ns) == 0 {
		return nil, nil
	}

	indentLevel := rule.indentLevel
	if indentLevel == 0 {
		indentLevel = DefaultIndentLevel
	}

	newNodes := make([]Node, 0, len(ns))

	var gcd func(a, b int) int
	gcd = func(a, b int) int {
		if b == 0 {
			return a
		}
		return gcd(b, a%b)
	}

	counts := make([]int, len(ns))
	var gcdValue int
	for idx, n := range ns {
		var count int
		for _, s := range n.Text() {
			if s == ' ' {
				count++
			} else if s == '\t' {
				count += indentLevel
			} else {
				break
			}
		}
		counts[idx] = count
		gcdValue = gcd(gcdValue, count)
	}

	if gcdValue == 0 {
		return ns, nil
	}

	for idx, n := range ns {
		count := counts[idx]
		txt := n.Text()
		txt = strings.ReplaceAll(txt, "\t", strings.Repeat(" ", indentLevel))
		txt = strings.TrimPrefix(txt, strings.Repeat(" ", count))
		txt = strings.Repeat(" ", (count/gcdValue)*indentLevel) + txt
		newNodes = append(newNodes, &node{txt})
	}

	return newNodes, nil
}
