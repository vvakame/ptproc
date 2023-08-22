package ptproc

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"

	"github.com/goccy/go-yaml"
	"go.opentelemetry.io/otel"
)

var _ slog.LogValuer = (*Config)(nil)
var _ slog.LogValuer = (*MapfileDirective)(nil)
var _ slog.LogValuer = (*MaprangeDirective)(nil)

type Config struct {
	Mapfile  *MapfileDirective  `yaml:"mapfile"`
	Maprange *MaprangeDirective `yaml:"maprange"`
}

type MapfileDirective struct {
	StartRegExp          string `yaml:"startRegExp"`
	EndRegExp            string `yaml:"endRegExp"`
	DisableRewriteIndent bool   `yaml:"disableRewriteIndent"`
	IndentWidth          int    `yaml:"indentWidth"`
	DefaultSkip          int    `yaml:"defaultSkip"`
}

type MaprangeDirective struct {
	StartRegExp          string `yaml:"startRegExp"`
	EndRegExp            string `yaml:"endRegExp"`
	DisableDedent        bool   `yaml:"disableDedent"`
	DisableRewriteIndent bool   `yaml:"disableRewriteIndent"`
	IndentWidth          int    `yaml:"indentWidth"`
	DefaultSkip          int    `yaml:"defaultSkip"`
}

func LoadConfig(ctx context.Context, filePath string) (_ *Config, err error) {
	ctx, span := otel.Tracer("ptproc").Start(ctx, "LoadConfig")
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	err = yaml.UnmarshalContext(ctx, b, cfg)
	if err != nil {
		return nil, err
	}

	err = cfg.fillByDefault()
	if err != nil {
		return nil, err
	}

	slog.DebugContext(ctx, "config file loaded", "config", cfg)

	return cfg, nil
}

func (cfg *Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("mapfile", cfg.Mapfile),
		slog.Any("maprange", cfg.Maprange),
	)
}

func (cfg *Config) fillByDefault() error {
	if cfg.Mapfile == nil {
		cfg.Mapfile = &MapfileDirective{
			StartRegExp:          "",
			EndRegExp:            "",
			DisableRewriteIndent: false,
			IndentWidth:          0,
			DefaultSkip:          0,
		}
	}
	if cfg.Mapfile.StartRegExp == "" {
		cfg.Mapfile.StartRegExp = DefaultMapfileStartRegEx.String()
	} else {
		re, err := regexp.Compile(cfg.Mapfile.StartRegExp)
		if err != nil {
			return fmt.Errorf("mapfile start regexp compile failed: %w", err)
		}
		if len(re.SubexpNames()) != 2 {
			return fmt.Errorf("mapfile start regexp doesn't satisfied restriction")
		}
	}
	if cfg.Mapfile.EndRegExp == "" {
		cfg.Mapfile.EndRegExp = DefaultMapfileEndRegEx.String()
	} else {
		_, err := regexp.Compile(cfg.Mapfile.EndRegExp)
		if err != nil {
			return fmt.Errorf("mapfile end regexp compile failed: %w", err)
		}
	}
	if cfg.Mapfile.IndentWidth == 0 {
		cfg.Mapfile.IndentWidth = 2
	}

	if cfg.Maprange == nil {
		cfg.Maprange = &MaprangeDirective{
			StartRegExp:          "",
			EndRegExp:            "",
			DisableDedent:        false,
			DisableRewriteIndent: false,
			IndentWidth:          0,
			DefaultSkip:          0,
		}
	}
	if cfg.Maprange.StartRegExp == "" {
		cfg.Maprange.StartRegExp = DefaultMaprangeStartRegEx.String()
	} else {
		re, err := regexp.Compile(cfg.Maprange.StartRegExp)
		if err != nil {
			return fmt.Errorf("maprange start regexp compile failed: %w", err)
		}
		if len(re.SubexpNames()) != 2 {
			return fmt.Errorf("maprange start regexp doesn't satisfied restriction")
		}
	}
	if cfg.Maprange.EndRegExp == "" {
		cfg.Maprange.EndRegExp = DefaultMaprangeEndRegEx.String()
	} else {
		_, err := regexp.Compile(cfg.Maprange.EndRegExp)
		if err != nil {
			return fmt.Errorf("maprange end regexp compile failed: %w", err)
		}
	}
	if cfg.Maprange.IndentWidth == 0 {
		cfg.Maprange.IndentWidth = 2
	}

	return nil
}

func (cfg *Config) ToProcessorConfig(ctx context.Context) (_ *ProcessorConfig, err error) {
	var rules []Rule
	{
		var mapfileStartRegExp *regexp.Regexp
		if v := cfg.Mapfile.StartRegExp; v != "" {
			mapfileStartRegExp, err = regexp.Compile(v)
			if err != nil {
				return nil, fmt.Errorf("mapfile.startRegExp compile failed: %w", err)
			}
		}
		var mapfileEndRegExp *regexp.Regexp
		if v := cfg.Mapfile.EndRegExp; v != "" {
			mapfileEndRegExp, err = regexp.Compile(v)
			if err != nil {
				return nil, fmt.Errorf("mapfile.endRegExp compile failed: %w", err)
			}
		}

		var embedRules []Rule
		if !cfg.Mapfile.DisableRewriteIndent {
			rule, err := NewReindentRule(&ReindentRuleConfig{
				IndentLevel: cfg.Mapfile.IndentWidth,
			})
			if err != nil {
				return nil, err
			}

			embedRules = append(embedRules, rule)
		}

		rule, err := NewMapfileRule(&MapfileRuleConfig{
			StartRegExp: mapfileStartRegExp,
			EndRegExp:   mapfileEndRegExp,
			DefaultSkip: cfg.Mapfile.DefaultSkip,
			EmbedRules:  embedRules,
		})
		if err != nil {
			return nil, err
		}

		rules = append(rules, rule)
	}

	{
		var maprangeStartRegExp *regexp.Regexp
		if v := cfg.Maprange.StartRegExp; v != "" {
			maprangeStartRegExp, err = regexp.Compile(v)
			if err != nil {
				return nil, fmt.Errorf("maprange.startRegExp compile failed: %w", err)
			}
		}
		var maprangeEndRegExp *regexp.Regexp
		if v := cfg.Maprange.EndRegExp; v != "" {
			maprangeEndRegExp, err = regexp.Compile(v)
			if err != nil {
				return nil, fmt.Errorf("mapfile.endRegExp compile failed: %w", err)
			}
		}

		var embedRules []Rule
		if !cfg.Maprange.DisableDedent {
			rule, err := NewDedentRule(&DedentRuleConfig{
				SpaceRegExp: nil,
			})
			if err != nil {
				return nil, err
			}

			embedRules = append(embedRules, rule)
		}
		if !cfg.Maprange.DisableRewriteIndent {
			rule, err := NewReindentRule(&ReindentRuleConfig{
				IndentLevel: cfg.Maprange.IndentWidth,
			})
			if err != nil {
				return nil, err
			}

			embedRules = append(embedRules, rule)
		}

		rule, err := NewMaprangeRule(&MaprangeRuleConfig{
			StartRegExp: maprangeStartRegExp,
			EndRegExp:   maprangeEndRegExp,
			DefaultSkip: cfg.Maprange.DefaultSkip,
			EmbedRules:  embedRules,
		})
		if err != nil {
			return nil, err
		}

		rules = append(rules, rule)
	}

	procCfg := &ProcessorConfig{
		Rules: rules,
	}

	return procCfg, nil
}

func (d *MapfileDirective) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("startRegExp", d.StartRegExp),
		slog.String("endRegExp", d.EndRegExp),
		slog.Bool("disableRewriteIndent", d.DisableRewriteIndent),
		slog.Int("indentWidth", d.IndentWidth),
		slog.Int("defaultSkip", d.DefaultSkip),
	)
}

func (d *MaprangeDirective) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("startRegExp", d.StartRegExp),
		slog.String("endRegExp", d.EndRegExp),
		slog.Bool("disableDedent", d.DisableDedent),
		slog.Bool("disableRewriteIndent", d.DisableRewriteIndent),
		slog.Int("indentWidth", d.IndentWidth),
		slog.Int("defaultSkip", d.DefaultSkip),
	)
}
