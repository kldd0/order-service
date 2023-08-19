package config

import (
	"os"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const configFile = "data/config.yaml"

type Config struct {
	DSN        string `yaml:"dsn"`
	HTTPServer `yaml:"http_server"`
}

type HTTPServer struct {
	Address string        `yaml:"address"`
	Timeout time.Duration `yaml:"timeout"`
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

func (s Service) DSN() string {
	return s.config.DSN
}

func (s Service) HTTPAddr() string {
	return s.config.HTTPServer.Address
}

func (s Service) Timeout() time.Duration {
	return s.config.HTTPServer.Timeout
}
