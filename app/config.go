package app

import (
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/harrybrwn/config"
	"github.com/spf13/pflag"
)

// Config is the application config struct
type Config struct {
	Host   string `config:"host" default:"0.0.0.0"`
	Port   int64  `config:"port" default:"8080"`
	Mode   string `config:"mode" default:"debug"`
	Secret []byte `config:"secret" env:"JWT_SECRET"`

	Database DatabaseConfig `config:"database"`
}

// DatabaseConfig is the part of the config struct that
// handles database info
type DatabaseConfig struct {
	Driver   string `config:"driver"`
	Host     string `config:"host" default:"localhost"`
	Port     int    `config:"port" default:"5432" env:"POSTGRES_PORT"`
	User     string `config:"user"`
	Password string `config:"password" env:"POSTGRES_PASSWORD"`

	// Database name or database filename
	Name string `config:"name"`
	SSL  string `config:"ssl" default:"disable"`
}

// Init sets up command line flags and parses command line args and gets config defaults
func (c *Config) Init() error {
	flag := pflag.NewFlagSet("api", pflag.ContinueOnError)

	flag.StringVar(&c.Host, "host", c.Host, "server host address")
	flag.Int64Var(&c.Port, "port", c.Port, "server port")
	flag.StringVar(&c.Mode, "mode", c.Mode, "change the gin server mode '(debug|release)'")

	flag.StringVar(&c.Database.Driver, "driver", c.Database.Driver, "database driver '[postgres|sqlite]'")
	flag.StringVar(&c.Database.Host, "db-host", c.Database.Host, "database remote host")
	flag.IntVar(&c.Database.Port, "db-port", c.Database.Port, "database remote port")
	flag.StringVar(&c.Database.Name, "name", c.Database.Name, "database name or filename")

	flag.StringVar(&c.Database.User, "user", c.Database.User, "database username")
	flag.StringVar(&c.Database.Password, "pw", c.Database.Password, "database user password")
	flag.StringVar(&c.Database.SSL, "db-ssl", c.Database.SSL, "toggle database ssl")

	flag.SortFlags = false
	switch err := flag.Parse(os.Args[1:]); err {
	case nil:
		break
	case pflag.ErrHelp:
		os.Exit(0)
	default:
		return err
	}
	return config.InitDefaults()
}

// GetDSN builds the database dns from the database config parameters
func (c *Config) GetDSN() string {
	return c.Database.GetDSN()
}

// GetDSN builds the database dns from the database config parameters
func (dbc *DatabaseConfig) GetDSN() string {
	switch dbc.Driver {
	case "postgres":
		return fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			dbc.Host, dbc.Port, dbc.User, dbc.Password, dbc.Name, dbc.SSL,
		)
	case "sqlite3":
		if !exists(dbc.Name) {
			return fmt.Sprintf("file:%s.sqlite", dbc.Name)
		}
		return fmt.Sprintf("file:%s", dbc.Name)
	default:
		panic(fmt.Sprintf("unknown database driver %s\n", dbc.Driver))
	}
}

// Address formats the server address:port from the app config
func (c *Config) Address() string {
	return net.JoinHostPort(
		config.GetString("host"),
		strconv.FormatInt(int64(config.GetInt("port")), 10),
	)
}

func exists(f string) bool {
	_, err := os.Stat(f)
	return !os.IsNotExist(err)
}

func statusColor(status int) string {
	var id int
	switch {
	case status == 0:
		id = 0
	case status < 300:
		id = 32
	case status < 400:
		id = 34
	case status < 500:
		id = 33
	case status < 600:
		id = 31
	default:
		id = 0
	}
	return fmt.Sprintf("\033[%d;1m", id)
}
