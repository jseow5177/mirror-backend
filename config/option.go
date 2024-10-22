package config

type Option struct {
	LogLevel   string
	ConfigPath string
	Port       int
}

func NewOptions() *Option {
	return &Option{
		LogLevel:   LogLevelDebug,
		ConfigPath: "./bin/config-test.json",
		Port:       9090,
	}
}
