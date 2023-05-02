package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

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
			&cli.BoolFlag{
				Name:    "replace",
				Usage:   "",
				Aliases: []string{"r"},
			},
		},
		Action: func(cCtx *cli.Context) error {
			ctx := cCtx.Context

			useReplace := cCtx.Bool("replace")

			fmt.Println(useReplace)

			proc, err := ptproc.NewProcessor(nil)
			if err != nil {
				return err
			}

			filePaths := cCtx.Args().Slice()
			if len(filePaths) == 0 {
				return errors.New("no files specified")
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
