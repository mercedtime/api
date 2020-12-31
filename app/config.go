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
	Host     string         `config:"host,shorthand=H,usage=server host" default:"0.0.0.0"`
	Port     int64          `config:"port,shorthand=P,usage=server port" default:"8080"`
	Mode     string         `config:"mode,usage=set the gin mode ('debug'|'release')" default:"debug"`
	Secret   string         `config:"secret,notflag" env:"JWT_SECRET"`
	Database DatabaseConfig `config:"db" yaml:"db"`
}

// DatabaseConfig is the part of the config struct that
// handles database info
type DatabaseConfig struct {
	Driver   string `config:"driver,usage=database driver name"`
	Host     string `config:"host,shorthand=h" default:"localhost"`
	Port     int    `config:"port,shorthand=p" default:"5432" env:"POSTGRES_PORT"`
	User     string `config:"user,shorthand=U"`
	Password string `config:"password" env:"POSTGRES_PASSWORD"`
	// Database name or database filename
	Name string `config:"name,shorthand=d,usage=name of the database"`
	SSL  string `config:"ssl" default:"disable"`
}

// Init sets up command line flags and parses command line args and gets config defaults
func (c *Config) Init() error {
	flag := pflag.NewFlagSet("api", pflag.ContinueOnError)
	config.BindToPFlagSet(flag)
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
