package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DSN string `yaml:"dsn"`
	Dir string `yaml:"migrationsDir"`
}

var ErrMissingConfigRequired = errors.New("missing config required: dsn or migrations_dir")

func NewConfig(flagDSN, flagDir, flagConfigFilePath string) (*Config, error) {
	conf := &Config{
		DSN: flagDSN,
		Dir: flagDir,
	}

	if conf.DSN != "" && conf.Dir != "" {
		return conf, nil
	}

	if flagConfigFilePath != "" {
		data, err := os.ReadFile(flagConfigFilePath)
		if err != nil {
			return nil, err
		}

		fileConfig := &Config{}

		err = yaml.Unmarshal(data, fileConfig)
		if err != nil {
			return nil, err
		}

		if conf.DSN == "" {
			conf.DSN = fileConfig.DSN
		}
		if conf.Dir == "" {
			conf.Dir = fileConfig.Dir
		}
	}

	if conf.DSN != "" && conf.Dir != "" {
		return conf, nil
	}

	if conf.DSN == "" {
		conf.DSN = os.Getenv("DB_DSN")
	}
	if conf.Dir == "" {
		conf.Dir = os.Getenv("MIGRATIONS_DIR")
	}

	if conf.DSN != "" && conf.Dir != "" {
		return conf, nil
	}

	return nil, ErrMissingConfigRequired
}
