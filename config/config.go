package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Backend struct {
	URL string `yaml:"url"`
}

type ClientLimit struct {
	Capacity   int `yaml:"capacity"`
	RatePerSec int `yaml:"rate_per_sec"`
}

type Config struct {
	Listen    string                 `yaml:"listen"`
	Backends  []Backend              `yaml:"backends"`
	Clients   map[string]ClientLimit `yaml:"clients"`
	CheckFreq int                    `yaml:"healthcheck_frequency"` // сек
}

func LoadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
