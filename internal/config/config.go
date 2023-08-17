package config

import (
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const configFile = "data/config.yaml"

type Config struct {
	Port  int    `yaml:"port"`
	DBuri string `yaml:"database_url"`
}

type Service struct {
	config Config
}

func New() (*Service, error) {
	var s *Service = &Service{}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, errors.Wrap(err, "Service.New: failed reading config file")
	}

	err = yaml.Unmarshal(data, &s.config)
	if err != nil {
		return nil, errors.Wrap(err, "Service.New: failed parsing yaml file")
	}

	return s, nil
}

func (s Service) Port() int {
	return s.config.Port
}

func (s Service) DBuri() string {
	return s.config.DBuri
}
