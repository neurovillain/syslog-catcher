package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Config - application configuration parameters.
type Config struct {
	Log struct {
		Level string `yaml:"level"`
		File  string `yaml:"file"`
	} `yaml:"log"`
	Syslog struct {
		Listen    string   `yaml:"listen"`
		Templates []string `yaml:"templates"`
		BufSize   int      `yaml:"buf_size"`
	} `yaml:"syslog"`
	GRPC struct {
		Listen string `yaml:"listen"`
	} `yaml:"grpc"`
}

// isValid - сheck current configuration state. returns nil if no errors are found.
func (c *Config) isValid() error {
	if len(c.Log.Level) == 0 {
		return fmt.Errorf("log level are not set")
	}
	if len(c.Log.File) == 0 {
		return fmt.Errorf("log output file are not set")
	}
	if len(c.Syslog.Listen) == 0 {
		return fmt.Errorf("syslog listen port are not set")
	}
	if len(c.Syslog.Templates) == 0 {
		return fmt.Errorf("no parsing templates are set")
	}
	if len(c.GRPC.Listen) == 0 {
		return fmt.Errorf("grpc listen port are not set")
	}

	return nil
}

// ParseFile - load configuration parameters from file. name - path to yaml config file,
// returns - parsed config struct or error, if occured.
func ParseFile(name string) (*Config, error) {
	buf, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read cfg file err - %v", err)
	}

	cfg := &Config{}
	err = yaml.Unmarshal(buf, cfg)
	if err != nil {
		return nil, fmt.Errorf("parse cfg data err - %v", err)
	}

	if err = cfg.isValid(); err != nil {
		return nil, fmt.Errorf("check cfg err - %v", err)
	}

	return cfg, nil
}
