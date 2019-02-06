package main

import "github.com/BurntSushi/toml"

type config struct {
	EnableCommands []string `toml:"enable_commands"`
}

// GetEnableCommands returns command list specified in configuration file
func GetEnableCommands(path string) ([]string, error) {
	var c config
	if _, err := toml.DecodeFile(path, &c); err != nil {
		return []string{}, err
	}
	return c.EnableCommands, nil
}
