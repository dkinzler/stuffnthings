package main

import (
	"backup/internal/script"
	"backup/internal/ui"
	"io"

	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v2"
)

type args struct {
	config     string
	log        string
	disableLog bool
}

func main() {
	app := &cli.App{
		Name:  "backup",
		Usage: "backup your stuff",
		Commands: []*cli.Command{
			{
				Name:  "tui",
				Usage: "use interactive terminal UI",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "log",
						Aliases: []string{"l"},
						Value:   "log.txt",
						Usage:   "log to file",
					},
					&cli.BoolFlag{
						Name:  "disableLog",
						Value: false,
						Usage: "disable logging",
					},
				},
				Action: func(cCtx *cli.Context) error {
					args := args{
						config:     cCtx.String("config"),
						log:        cCtx.String("log"),
						disableLog: cCtx.Bool("disableLog"),
					}
					return runUI(args)
				},
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "",
				Usage:   "config file",
			},
		},
		Action: func(cCtx *cli.Context) error {
			args := args{
				config: cCtx.String("config"),
			}
			return run(args)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(args args) error {
	script.Backup(args.config)
	return nil
}

// Note: the log package is safe to use with multiple goroutines, fmt is not and might produce mixed output.
func runUI(args args) error {
	if args.disableLog {
		// by default log writes to stdout and would interfere with our TUI
		log.SetOutput(io.Discard)
	} else {
		f, err := tea.LogToFile(args.log, "")
		if err != nil {
			return err
		}
		defer f.Close()
	}

	p := tea.NewProgram(ui.NewModel(args.config), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
