package main

import (
	"os"

	"github.com/codegangsta/cli"
)

func main() {
	app := cli.NewApp()

	app.Commands = []cli.Command{
		{
			Name:        "project",
			Usage:       "Actions on projects",
			Subcommands: []cli.Command{createProjectCommand(), listProjectsCommand()},
		}, {
			Name:        "job",
			Usage:       "Actions on projects",
			Subcommands: []cli.Command{startJobCommand()},
		},
	}
	app.Run(os.Args)
}