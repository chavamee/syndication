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

package config

import (
	"bufio"
	"github.com/BurntSushi/toml"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	SystemConfigPath = "/etc/syndication/config.toml"
	UserConfigPath   = "%s/syndication/config.toml"
)

type (
	Server struct {
		AuthSecret            string `toml:"auth_secret"`
		AuthSecreteFilePath   string `toml:"auth_secret_file_path"`
		EnableRequestLogs     bool   `toml:"enable_http_requests_log"`
		EnablePanicPrintStack bool   `toml:"enable_panic_print_stack"`
	}

	Database struct {
		Type       string `toml:"-"`
		Enable     bool
		Connection string
	}

	Sync struct {
		SyncTime     time.Time     `toml:"time"`
		SyncInterval time.Duration `toml:"interval"`
	}

	Admin struct {
		AdminSocketPath string `toml:"socket_path"`
	}

	Config struct {
		Database  `toml:"-"`
		Databases map[string]Database `toml:"database"`
		Server    Server
		Sync      Sync
		Admin     Admin
	}
)

var (
	defaultConfig = Config{
		Databases: map[string]Database{
			"sqlite": {
				Type:       "sqlite3",
				Connection: "/var/syndication/db/syndication.db",
			}},

		Server: Server{
			EnableRequestLogs:     false,
			EnablePanicPrintStack: true,
			AuthSecret:            "",
			AuthSecreteFilePath:   "",
		},

		Sync: Sync{
			SyncInterval: time.Minute * 15,
		},
	}
)

func (c *Config) getSecretFromFile(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return InvalidFieldValue{"Invalid secrete file path"}
	}

	c.Server.AuthSecreteFilePath = absPath

	fi, err := os.Open(c.Server.AuthSecreteFilePath)
	if err != nil {
		return FileSystemError{"Could not read secrete file"}
	}

	r := bufio.NewReader(fi)
	buf := make([]byte, 512)
	if _, err := r.Read(buf); err != nil && err != io.EOF {
		return FileSystemError{"Could not read secrete file"}
	}

	c.Server.AuthSecret = string(buf)

	if err := fi.Close(); err != nil {
		return FileSystemError{"Could not close secrete file"}
	}

	return nil
}

func (c *Config) checkSQLiteConfig() error {
	path := c.Databases["sqlite"].Connection
	if path == "" {
		return InvalidFieldValue{"DB path cannot be empty"}
	}

	if !filepath.IsAbs(path) {
		return InvalidFieldValue{"DB path must be absolute"}
	}

	return nil
}

func NewConfig(path string) (config Config, err error) {
	_, err = os.Stat(path)

	if err == nil {
		_, err = toml.DecodeFile(path, &config)
	}

	if err != nil {
		err = ParsingError{"Unable to parse error"}
		return
	}

	if config.Server.AuthSecreteFilePath != "" {
		err = config.getSecretFromFile(config.Server.AuthSecreteFilePath)
		if err != nil {
			return
		}
	} else if config.Server.AuthSecret == "" {
		err = InvalidFieldValue{"Auth secret should not be empty"}
		return
	}

	if len(config.Databases) > 1 {
		err = InvalidFieldValue{"Can only have one database definition"}
		return
	}

	if len(config.Databases) == 0 {
		err = InvalidFieldValue{"Configuration requires a database definition"}
	}

	for dbType, db := range config.Databases {
		if dbType == "sqlite" {
			if !db.Enable {
				continue
			}

			err = config.checkSQLiteConfig()
			if err != nil {
				return
			}

			config.Database.Connection = config.Databases["sqlite"].Connection
			config.Database.Type = "sqlite3"
		} else if dbType == "mysql" {
		} else if dbType == "postgres" {
		} else {
			err = InvalidFieldValue{"Database type cannot be empty"}
			return
		}
	}

	if config.Database == (Database{}) {
		err = InvalidFieldValue{"Database not defined or not enabled"}
	}
	return
}

func NewDefaultConfig() Config {
	return defaultConfig
}
