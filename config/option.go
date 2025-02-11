package config

import (
	"os"
	"strconv"
)

type Option struct {
	LogLevel   string
	ConfigPath string
	Port       int
}

func NewOptions() *Option {
	opt := &Option{
		LogLevel:   LogLevelDebug,
		ConfigPath: "./bin/config-test.json",
		Port:       DefaultPort,
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		opt.LogLevel = logLevel
	}

	if configPath := os.Getenv("CONFIG_PATH"); configPath != "" {
		opt.ConfigPath = configPath
	}

	if serverPort := os.Getenv("PORT"); serverPort != "" {
		if port, err := strconv.Atoi(serverPort); err == nil {
			opt.Port = port
		}
	}

	return opt
}
