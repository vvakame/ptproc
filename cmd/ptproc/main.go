package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/urfave/cli/v2"
	"github.com/vvakame/ptproc"
	"golang.org/x/exp/slog"
	"golang.org/x/sync/errgroup"
)

func main() {
	err := realMain()
	if err != nil {
		log.Fatal(err)
	}
}

func realMain() error {
	ctx := context.Background()

	setDefaultLoggerWithLevel(slog.LevelInfo)

	app := &cli.App{
		Name:  "ptproc",
		Usage: "plain text preprocessor",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "logLevel",
				Usage: "--logLevel (debug|info|warn|error)",
				Action: func(cCtx *cli.Context, logLevel string) error {
					ctx := cCtx.Context

					var leveler slog.Leveler
					switch logLevel {
					case "debug":
						leveler = slog.LevelDebug
					case "info":
						leveler = slog.LevelInfo
					case "warn":
						leveler = slog.LevelWarn
					case "error":
						leveler = slog.LevelError
					default:
						return fmt.Errorf("unknown logLevel: %s", logLevel)
					}

					setDefaultLoggerWithLevel(leveler)
					slog.DebugCtx(ctx, "set default log level", slog.String("logLevel", logLevel))

					return nil
				},
			},
			&cli.StringFlag{
				Name:        "config",
				Usage:       "syntax config file path",
				DefaultText: "./ptproc.yaml",
				Aliases:     []string{"c"},
			},
			&cli.BoolFlag{
				Name:    "replace",
				Usage:   "write back result to source file instead of stdout",
				Aliases: []string{"r"},
			},
			&cli.StringFlag{
				Name:    "glob",
				Usage:   "specify target file by glob pattern. see https://pkg.go.dev/path/filepath#Glob",
				Aliases: []string{"g"},
			},
		},
		Action: func(cCtx *cli.Context) error {
			ctx := cCtx.Context

			configFilePath := cCtx.String("config")
			configFileSpecified := true
			if configFilePath == "" {
				configFilePath = "ptproc.yaml"
				configFileSpecified = false
			}
			useReplace := cCtx.Bool("replace")
			globPattern := cCtx.String("glob")

			var cfg *ptproc.ProcessorConfig
			if rawCfg, err := ptproc.LoadConfig(ctx, configFilePath); !configFileSpecified && errors.Is(err, os.ErrNotExist) {
				slog.DebugCtx(ctx, "ptproc.yaml is not exists. ignored")
				cfg = nil
			} else if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("failed to load config file: %s, : %w", configFilePath, err)
			} else if err != nil {
				return err
			} else {
				var rules []ptproc.Rule

				slog.DebugCtx(ctx, "config file loaded", "config", rawCfg)

				{
					var mapfileStartRegExp *regexp.Regexp
					if v := rawCfg.Mapfile.StartRegExp; v != "" {
						mapfileStartRegExp, err = regexp.Compile(v)
						if err != nil {
							return fmt.Errorf("mapfile.startRegExp compile failed: %w", err)
						}
					}
					var mapfileEndRegExp *regexp.Regexp
					if v := rawCfg.Mapfile.EndRegExp; v != "" {
						mapfileEndRegExp, err = regexp.Compile(v)
						if err != nil {
							return fmt.Errorf("mapfile.endRegExp compile failed: %w", err)
						}
					}

					var embedRules []ptproc.Rule
					if !rawCfg.DisableRewriteIndent {
						rule, err := ptproc.NewReindentRule(&ptproc.ReindentRuleConfig{
							IndentLevel: rawCfg.IndentWidth,
						})
						if err != nil {
							return err
						}

						embedRules = append(embedRules, rule)
					}

					rule, err := ptproc.NewMapfileRule(&ptproc.MapfileRuleConfig{
						StartRegExp: mapfileStartRegExp,
						EndRegExp:   mapfileEndRegExp,
						EmbedRules:  embedRules,
					})
					if err != nil {
						return err
					}

					rules = append(rules, rule)
				}

				{
					var maprangeStartRegExp *regexp.Regexp
					if v := rawCfg.Maprange.StartRegExp; v != "" {
						maprangeStartRegExp, err = regexp.Compile(v)
						if err != nil {
							return fmt.Errorf("maprange.startRegExp compile failed: %w", err)
						}
					}
					var maprangeEndRegExp *regexp.Regexp
					if v := rawCfg.Maprange.EndRegExp; v != "" {
						maprangeEndRegExp, err = regexp.Compile(v)
						if err != nil {
							return fmt.Errorf("mapfile.endRegExp compile failed: %w", err)
						}
					}

					var embedRules []ptproc.Rule
					if !rawCfg.DisableRewriteIndent {
						rule, err := ptproc.NewDedentRule(&ptproc.DedentRuleConfig{
							SpaceRegExp: nil,
						})
						if err != nil {
							return err
						}

						embedRules = append(embedRules, rule)
					}
					if !rawCfg.DisableRewriteIndent {
						rule, err := ptproc.NewReindentRule(&ptproc.ReindentRuleConfig{
							IndentLevel: rawCfg.IndentWidth,
						})
						if err != nil {
							return err
						}

						embedRules = append(embedRules, rule)
					}

					rule, err := ptproc.NewMaprangeRule(&ptproc.MaprangeRuleConfig{
						StartRegExp: maprangeStartRegExp,
						EndRegExp:   maprangeEndRegExp,
						EmbedRules:  embedRules,
					})
					if err != nil {
						return err
					}

					rules = append(rules, rule)
				}

				cfg = &ptproc.ProcessorConfig{
					Rules: rules,
				}
			}

			slog.DebugCtx(ctx, "start processing", slog.Bool("replace", useReplace), slog.String("glob", globPattern))

			var filePaths []string

			if fs := cCtx.Args().Slice(); len(fs) != 0 {
				filePaths = append(filePaths, fs...)
			}

			if globPattern != "" {
				fs, err := filepath.Glob(globPattern)
				if err != nil {
					return err
				}

				filePaths = append(filePaths, fs...)
			}

			if len(filePaths) == 0 {
				return errors.New("no files specified")
			}

			slog.DebugCtx(ctx, "target files", "filePaths", filePaths)

			proc, err := ptproc.NewProcessor(cfg)
			if err != nil {
				return err
			}

			var eg errgroup.Group

			for _, s := range filePaths {
				s := s

				if useReplace {
					eg.Go(func() error {
						slog.DebugCtx(ctx, "replace file", slog.String("file", s))

						result, err := proc.ProcessFile(ctx, s)
						if err != nil {
							return err
						}

						err = os.WriteFile(s, []byte(result), 0o644)
						if err != nil {
							return err
						}

						slog.InfoCtx(ctx, "file has been replaced", slog.String("file", s))

						return nil
					})

				} else {
					result, err := proc.ProcessFile(ctx, s)
					if err != nil {
						return err
					}

					fmt.Println(result)
				}
			}

			err = eg.Wait()
			if err != nil {
				return err
			}

			return nil
		},
	}

	if err := app.RunContext(ctx, os.Args); err != nil {
		return err
	}

	return nil
}

func setDefaultLoggerWithLevel(level slog.Leveler) {
	h := slog.HandlerOptions{
		Level: level,
	}.NewTextHandler(os.Stderr)
	logger := slog.New(h)
	slog.SetDefault(logger)
}
