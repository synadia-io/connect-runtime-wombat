package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"slices"
)

func LoadConfig(base *Config, path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !fi.IsDir() {
		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read config file %s: %w", path, err)
		}

		var cfg Config
		if err := yaml.Unmarshal(b, &cfg); err != nil {
			return fmt.Errorf("failed to unmarshal config file %s: %w", path, err)
		}

		base.Merge(&cfg)
	} else {
		des, err := os.ReadDir(path)
		if err != nil {
			return fmt.Errorf("failed to read config directory %s: %w", path, err)
		}

		for _, d := range des {
			if err := LoadConfig(base, path+"/"+d.Name()); err != nil {
				return err
			}
		}
	}

	return nil
}

type Config struct {
	Components map[string][]string `yaml:"components,omitempty"`
}

func (c *Config) Contains(kind string, name string) bool {
	if c.Components == nil {
		return false
	}

	if _, ok := c.Components[kind]; !ok {
		return false
	}

	return slices.Contains(c.Components[kind], name)
}

func (c *Config) Merge(other *Config) {
	if other == nil || other.Components == nil {
		return
	}

	for kind, components := range other.Components {
		if c.Components == nil {
			c.Components = make(map[string][]string)
		}

		c.Components[kind] = append(c.Components[kind], components...)
	}
}
