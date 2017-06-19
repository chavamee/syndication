/*
  Copyright (C) 2017 Jorge Martinez Hernandez

  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU Affero General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU Affero General Public License for more details.

  You should have received a copy of the GNU Affero General Public License
  along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"os"

	"github.com/chavamee/syndication/admin"
	"github.com/chavamee/syndication/config"
	"github.com/chavamee/syndication/database"
	"github.com/chavamee/syndication/server"
	"github.com/chavamee/syndication/sync"
	"github.com/fatih/color"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()

	app.Name = "syndication"
	app.Usage = "A Super Simple RSS server"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config",
			Usage: "Path to configuration file to use",
		},
		cli.StringFlag{
			Name:  "socket",
			Usage: "Path to admin socket",
		},
		cli.BoolFlag{
			Name:  "admin",
			Usage: "Enable/Disable admin",
		},
		cli.BoolFlag{
			Name:  "sync",
			Usage: "Enable/Disable sync",
		},
	}

	app.Action = func(c *cli.Context) error {
		var conf config.Config
		var err error

		if c.String("config") == "" {
			conf = config.NewDefaultConfig()
		} else {
			conf, err = config.NewConfig(c.String("config"))
			if err != nil {
				color.Red(err.Error())
				return err
			}
		}

		dbImpl, err := database.NewDB(conf.Database.Type, conf.Database.Connection)
		if err != nil {
			return err
		}
		sync := sync.NewSync(&dbImpl)
		sync.Start()

		conf.Admin.AdminSocketPath = "/tmp/syndication"
		adminSocket, err := admin.NewAdminSocket(&dbImpl, conf.Admin.AdminSocketPath)
		if err != nil {
			return err
		}
		adminSocket.Start()

		defer adminSocket.Stop()

		server := server.NewServer(&dbImpl, &sync, conf.Server)
		server.Start()

		return err
	}

	app.Run(os.Args)
}
