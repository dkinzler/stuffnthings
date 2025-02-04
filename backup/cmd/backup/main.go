package main

import (
	"backup/internal"

	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "backup",
		Usage: "",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "log",
				Aliases: []string{"l"},
				Value:   "",
				Usage:   "log to file",
			},
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "",
				Usage:   "config file",
			},
		},
		Action: func(cCtx *cli.Context) error {
			logFile := cCtx.String("log")
			configFile := cCtx.String("config")
			return run(logFile, configFile)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(logFile, configFile string) error {
	if logFile != "" {
		f, err := tea.LogToFile(logFile, "log")
		if err != nil {
			return err
		}
		defer f.Close()
	}
	// TODO what happens otherwise, log will print to screen and fuck up our things? in that case should we set log to log nowhere
	// TODO we should probably add some log messages throughout for debugging? why not
	// can we set log levels wit log package?

	p := tea.NewProgram(internal.NewModel(configFile), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
