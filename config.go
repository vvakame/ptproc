package ptproc

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/goccy/go-yaml"
	"go.opentelemetry.io/otel"
	"golang.org/x/exp/slog"
)

var _ slog.LogValuer = (*Config)(nil)
var _ slog.LogValuer = (*MapfileDirective)(nil)
var _ slog.LogValuer = (*MaprangeDirective)(nil)

type Config struct {
	Mapfile              *MapfileDirective  `yaml:"mapfile"`
	Maprange             *MaprangeDirective `yaml:"maprange"`
	DisableRewriteIndent bool               `yaml:"disableRewriteIndent"`
	IndentWidth          int                `yaml:"indentWidth"`
}

type MapfileDirective struct {
	StartRegExp string `yaml:"startRegExp"`
	EndRegExp   string `yaml:"endRegExp"`
}

type MaprangeDirective struct {
	StartRegExp   string `yaml:"startRegExp"`
	EndRegExp     string `yaml:"endRegExp"`
	DisableDedent bool   `yaml:"disableDedent"`
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

	return cfg, nil
}

func (cfg *Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("mapfile", cfg.Mapfile),
		slog.Any("maprange", cfg.Maprange),
		slog.Bool("disableRewriteIndent", cfg.DisableRewriteIndent),
		slog.Int("indentWidth", cfg.IndentWidth),
	)
}

func (cfg *Config) fillByDefault() error {
	regexpHasGroups := func(re *regexp.Regexp, ss ...string) bool {
	OUTER:
		for _, s := range ss {
			for _, g := range re.SubexpNames() {
				if s == g {
					continue OUTER
				}
			}
			return false
		}
		return true
	}

	if cfg.Mapfile == nil {
		cfg.Mapfile = &MapfileDirective{
			StartRegExp: "",
			EndRegExp:   "",
		}
	}
	if cfg.Mapfile.StartRegExp == "" {
		cfg.Mapfile.StartRegExp = DefaultMapfileStartRegEx.String()
	} else {
		re, err := regexp.Compile(cfg.Mapfile.StartRegExp)
		if err != nil {
			return fmt.Errorf("mapfile start regexp compile failed: %w", err)
		}
		if !regexpHasGroups(re, "FilePath") {
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

	if cfg.Maprange == nil {
		cfg.Maprange = &MaprangeDirective{
			StartRegExp: "",
			EndRegExp:   "",
		}
	}
	if cfg.Maprange.StartRegExp == "" {
		cfg.Maprange.StartRegExp = DefaultMaprangeStartRegEx.String()
	} else {
		re, err := regexp.Compile(cfg.Maprange.StartRegExp)
		if err != nil {
			return fmt.Errorf("maprange start regexp compile failed: %w", err)
		}
		if !regexpHasGroups(re, "FilePath", "RangeName") {
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

	if cfg.IndentWidth == 0 {
		cfg.IndentWidth = 2
	}

	return nil
}

func (d *MapfileDirective) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("startRegExp", d.StartRegExp),
		slog.String("endRegExp", d.EndRegExp),
	)
}

func (d *MaprangeDirective) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("startRegExp", d.StartRegExp),
		slog.String("endRegExp", d.EndRegExp),
	)
}
