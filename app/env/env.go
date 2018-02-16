package env

import (
	"os"

	"github.com/lbryio/errors.go"

	e "github.com/caarlos0/env"
	"github.com/go-ini/ini"
)

type Config struct {
	Port        string `env:"PORT" envDefault:"8080"`
	MysqlDsn    string `env:"MYSQL_DSN"`
	LbrycrdURL  string `env:"LBRYCRD_CONNECT"`
	NewRelicKey string `env:"NEW_RELIC_KEY"`
}

// NewWithEnvVars creates an Config from environment variables
func NewWithEnvVars() (*Config, error) {
	cfg := &Config{}
	err := e.Parse(cfg)
	if err != nil {
		return nil, errors.Err(err)
	}

	if cfg.LbrycrdURL == "from_conf" {
		cfg.LbrycrdURL, err = getLbrycrdURLFromConfFile()
		if err != nil {
			return nil, err
		}
	}

	if cfg.MysqlDsn == "" {
		return nil, errors.Err("MYSQL_DSN env var required")
	}

	return cfg, nil
}

func getLbrycrdURLFromConfFile() (string, error) {
	if os.Getenv("HOME") == "" {
		return "", errors.Err("$HOME env var not set")
	}

	defaultConfFile := os.Getenv("HOME") + "/.lbrycrd/lbrycrd.conf"
	if _, err := os.Stat(defaultConfFile); os.IsNotExist(err) {
		return "", errors.Err("lbrycrd conf file not found")
	}

	cfg, err := ini.Load(defaultConfFile)
	if err != nil {
		return "", errors.Err(err)
	}

	section, err := cfg.GetSection("")
	if err != nil {
		return "", errors.Err(err)
	}

	username := section.Key("rpcuser").String()
	password := section.Key("rpcpassword").String()
	host := section.Key("rpchost").String()
	if host == "" {
		host = "localhost"
	}
	port := section.Key("rpcport").String()
	if port == "" {
		port = ":9245"
	} else {
		port = ":" + port
	}

	userpass := ""
	if username != "" || password != "" {
		userpass = username + ":" + password + "@"
	}

	return "rpc://" + userpass + host + port, nil
}
