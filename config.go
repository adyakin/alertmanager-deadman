package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Labels      map[string]string `yaml:"labels,flow"`
	Annotations map[string]string `yaml:"annotations,flow"`
}

func NewConfig(path string) (*Config, error) {
	config := &Config{}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Init new YAML decode
	d := yaml.NewDecoder(file)

	// Start YAML decoding from file
	if err := d.Decode(&config); err != nil {
		return nil, err
	}

	return config, nil
}
