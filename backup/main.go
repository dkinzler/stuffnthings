package main

import (
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
				Name:    "logFile",
				Aliases: []string{"l"},
				Value:   "log.txt",
				Usage:   "save log messages to file",
			},
		},
		Action: func(cCtx *cli.Context) error {
			logFile := cCtx.String("logFile")
			return run(logFile)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(logFile string) error {
	f, err := tea.LogToFile(logFile, "log")
	if err != nil {
		return err
	}
	defer f.Close()
	log.SetOutput(f)

	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
