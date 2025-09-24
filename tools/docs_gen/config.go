package main

import (
	"fmt"
	"os"
	"slices"

	"gopkg.in/yaml.v3"
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
	Components map[string][]string          `yaml:"components,omitempty"`
	Icons      map[string]map[string]string `yaml:"icons,omitempty"`
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

func (c *Config) GetIcon(kind string, name string) string {
	if c.Icons == nil {
		return ""
	}

	if kindIcons, ok := c.Icons[kind]; ok {
		if icon, ok := kindIcons[name]; ok {
			return icon
		}
	}

	return ""
}

func (c *Config) Merge(other *Config) {
	if other == nil {
		return
	}

	// Merge components
	if other.Components != nil {
		for kind, components := range other.Components {
			if c.Components == nil {
				c.Components = make(map[string][]string)
			}

			c.Components[kind] = append(c.Components[kind], components...)
		}
	}

	// Merge icons
	if other.Icons != nil {
		for kind, icons := range other.Icons {
			if c.Icons == nil {
				c.Icons = make(map[string]map[string]string)
			}
			if c.Icons[kind] == nil {
				c.Icons[kind] = make(map[string]string)
			}

			for name, icon := range icons {
				c.Icons[kind][name] = icon
			}
		}
	}
}
