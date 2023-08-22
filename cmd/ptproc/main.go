package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
	"github.com/vvakame/ptproc"
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
					slog.DebugContext(ctx, "set default log level", slog.String("logLevel", logLevel))

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
				slog.DebugContext(ctx, "ptproc.yaml is not exists. ignored")
				cfg = nil
			} else if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("failed to load config file: %s, : %w", configFilePath, err)
			} else if err != nil {
				return err
			} else {
				cfg, err = rawCfg.ToProcessorConfig(ctx)
				if err != nil {
					return err
				}
			}

			slog.DebugContext(ctx, "start processing", slog.Bool("replace", useReplace), slog.String("glob", globPattern))

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

			slog.DebugContext(ctx, "target files", "filePaths", filePaths)

			proc, err := ptproc.NewProcessor(cfg)
			if err != nil {
				return err
			}

			var eg errgroup.Group

			for _, s := range filePaths {
				s := s

				if useReplace {
					eg.Go(func() error {
						slog.DebugContext(ctx, "replace file", slog.String("file", s))

						result, err := proc.ProcessFile(ctx, s)
						if err != nil {
							return err
						}

						err = os.WriteFile(s, []byte(result), 0o644)
						if err != nil {
							return err
						}

						slog.InfoContext(ctx, "file has been replaced", slog.String("file", s))

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
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	logger := slog.New(h)
	slog.SetDefault(logger)
}
