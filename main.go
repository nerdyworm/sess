package main

import (
	"log"
	"os"

	"github.com/codegangsta/cli"
	"github.com/nerdyworm/sess/app"
	"github.com/nerdyworm/sess/queue"
	"github.com/nerdyworm/sess/repos"
	"github.com/nerdyworm/sess/workers"
)

func main() {
	log.SetFlags(log.Lshortfile)

	repos.Setup()
	defer repos.Shutdown()

	queue.Setup()
	defer queue.Shutdown()

	app.Setup()

	a := cli.NewApp()
	a.Name = "web"
	a.Usage = "run web server"
	a.Commands = []cli.Command{
		cli.Command{
			Name:        "web",
			Description: "run http server",
			Action: func(c *cli.Context) {
				app.Run()
			},
		},

		cli.Command{
			Name:        "workers",
			Description: "run workers",
			Action: func(c *cli.Context) {
				workers.Run()
			},
		},
	}

	a.Run(os.Args)
}
