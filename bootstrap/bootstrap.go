package bootstrap

import (
	"errors"
	"fmt"
	"os"

	"github.com/chavamee/syndication/config"

	"github.com/fatih/color"
)

type (
	Bootstrapper struct {
		config     config.Config
		advanced   bool
		configPath string
	}
)

func printHeader(str string) {
	color.Set(color.FgWhite, color.Bold)
	fmt.Println(str)
	color.Unset()
	color.Set(color.FgGreen, color.Bold)
	fmt.Println("====================")
	color.Unset()
}

func setupSQLite(conf *config.Database) error {
	printHeader("Configuring SQLite")

	fmt.Printf("Enter database file path (Default: %s): ", conf.Connection)
	color.Set(color.FgBlue, color.Bold)

	var path string
	num, err := fmt.Scanln(&path)
	if num == 1 && err == nil {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			path = path + "/syndication.db"
		}
		conf.Connection = path
	}

	color.Unset()

	conf.Type = "sqlite3"

	return nil
}

func getServerSecret(conf *config.Server) error {
	fmt.Println("1) Generate an auth secret")
	fmt.Println("2) Provide an auth secret")
	fmt.Println("3) Enter path to auth secret")

	fmt.Print("Select a strategy (Default: Generate secret): ")

	var selection int
	_, err := fmt.Scanf("%d", &selection)
	if err != nil {
		selection = 1
	}

	switch selection {
	case 1:
		// TODO: Generate a secret
		break
	case 2:
		fmt.Print("Enter auth secret: ")
		color.Set(color.FgYellow, color.Bold)

		var secret string
		var num int
		num, err = fmt.Scanln(&secret)
		if err != nil {
			if num < 1 {
				err = errors.New("secret cannot be empty")
			}
			return err
		}

		conf.AuthSecret = secret

		color.Unset()
		break
	case 3:
		fmt.Print("Enter auth secret file path: ")
		color.Set(color.FgBlue, color.Bold)

		var path string
		var num int
		num, err = fmt.Scanln(&path)
		if err != nil {
			if num < 1 {
				err = errors.New("path cannot be empty")
			}
			return err
		}

		color.Unset()

		var info os.FileInfo
		info, err = os.Stat(path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			err = errors.New("path must point to a regular file")
			return err
		}

		conf.AuthSecreteFilePath = path

		break
	}

	return nil
}

func (b *Bootstrapper) SetupDatabase() (conf config.Database, err error) {
	conf = config.DefaultDatabaseConfig

	printHeader("Configuring database")

	fmt.Println("1) mysql")
	fmt.Println("2) postgresssql")
	fmt.Println("3) sqlite3")

	fmt.Printf("Select a database type (Default: %s): ", config.DefaultDatabaseConfig.Type)

	var selection int
	_, err = fmt.Scanf("%d", &selection)
	if err != nil {
		selection = 3
	}

	fmt.Println()

	switch selection {
	case 3:
		err = setupSQLite(&conf)
		if err != nil {
			return
		}
		break
	}

	fmt.Println()

	return
}

func (b *Bootstrapper) SetupAdmin() (config.Admin, error) {
	conf := config.DefaultAdminConfig
	printHeader("Configuring admin")

	fmt.Printf("Enter admin socket path (Default: %s): ", config.DefaultAdminConfig.SocketPath)
	color.Set(color.FgBlue, color.Bold)

	var path string
	num, err := fmt.Scanln(&path)
	if num == 1 && err == nil {
		var info os.FileInfo
		info, err = os.Stat(path)
		if err != nil {
			return config.Admin{}, err
		}

		if info.IsDir() {
			path = path + "/syndication.admin"
		}

		conf.SocketPath = path
	}

	color.Unset()

	fmt.Println()

	return conf, nil
}

func (b *Bootstrapper) SetupServer() (config.Server, error) {
	conf := config.DefaultServerConfig

	printHeader("Configuring server")

	err := getServerSecret(&conf)
	if err != nil {
		return config.Server{}, err
	}

	fmt.Println()

	fmt.Printf("Enter an HTTP port (Default: %d): ", config.DefaultServerConfig.Port)

	color.Set(color.FgMagenta, color.Bold)

	var port int
	_, err = fmt.Scanf("%d", &port)
	if err == nil {
		conf.Port = port
	}

	color.Unset()

	fmt.Println()

	return conf, nil
}

func (b *Bootstrapper) Setup() (config.Config, error) {
	var err error
	b.config.Database, err = b.SetupDatabase()
	if err != nil {
		return config.Config{}, err
	}

	b.config.Admin, err = b.SetupAdmin()
	if err != nil {
		return config.Config{}, err
	}

	b.config.Server, err = b.SetupServer()
	if err != nil {
		return config.Config{}, err
	}

	b.config.Sync = config.DefaultSyncConfig

	return b.config, nil
}

func NewBootstrapper(configPath string, showAdvanced bool) *Bootstrapper {
	return &Bootstrapper{
		config:     config.NewEmptyConfig(configPath),
		configPath: configPath,
		advanced:   showAdvanced,
	}
}
