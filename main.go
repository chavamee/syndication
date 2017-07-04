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
	"fmt"
	"os"

	"github.com/chavamee/syndication/admin"
	"github.com/chavamee/syndication/bootstrap"
	"github.com/chavamee/syndication/config"
	"github.com/chavamee/syndication/database"
	"github.com/chavamee/syndication/server"
	"github.com/chavamee/syndication/sync"

	"github.com/fatih/color"
	"github.com/urfave/cli"
)

func findConfig() (config.Config, bool, error) {
	if _, err := os.Stat(config.SystemConfigPath); err == nil {
		conf, err := config.NewConfig(config.SystemConfigPath)
		if err != nil {
			color.Set(color.FgRed, color.Bold)
			fmt.Println("Failed to parse system configuration file")
			color.Unset()
			return config.Config{}, true, err
		}

		return conf, true, nil
	}

	userPath := fmt.Sprintf(config.UserConfigPath, os.Getenv("HOME"))
	if _, err := os.Stat(userPath); err == nil {
		conf, err := config.NewConfig(userPath)
		if err != nil {
			color.Set(color.FgRed, color.Bold)
			fmt.Println("Failed to parse user configuration file")
			color.Unset()
			return config.Config{}, true, err
		}
		return conf, true, nil
	}

	return config.Config{}, false, nil
}

func startApp(c *cli.Context) error {
	var conf config.Config

	color.Set(color.FgGreen, color.Bold)
	fmt.Println("Starting Syndication")
	fmt.Println()
	color.Unset()

	if c.String("config") == "" {
		sysConfig, found, err := findConfig()
		if found {
			if err != nil {
				return err
			}

			conf = sysConfig
		} else if c.Bool("skip-bootstrap") == true {
			conf = config.DefaultConfig
		} else {
			bs := bootstrap.NewBootstrapper("/tmp/synd.toml", false)
			var err error
			conf, err = bs.Setup()
			if err != nil {
				return err
			}

			fmt.Println("Saving configuration")
			err = conf.Save()
			if err != nil {
				return err
			}
		}
	} else {
		var err error
		conf, err = config.NewConfig(c.String("config"))
		if err != nil {
			return err
		}
	}

	db, err := database.NewDB(conf.Database.Type, conf.Database.Connection)
	if err != nil {
		return err
	}
	sync := sync.NewSync(db)
	sync.Start()

	admin, err := admin.NewAdmin(db, conf.Admin.SocketPath)
	if err != nil {
		return err
	}
	admin.Start()

	defer admin.Stop(true)

	server := server.NewServer(db, sync, conf.Server)
	server.Start()

	return err
}

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
			Name:  "skip-boostrap",
			Usage: "Skip bootstrapping",
		},
		cli.BoolFlag{
			Name:  "sync",
			Usage: "Enable/Disable sync",
		},
	}

	app.Action = startApp

	err := app.Run(os.Args)
	if err != nil {
		color.Red(err.Error())
	}
}
