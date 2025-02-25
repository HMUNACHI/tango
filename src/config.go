package tango

import (
	"github.com/spf13/viper"
)

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type TokensConfig struct {
	Approved []string `mapstructure:"approved"`
}

type TaskConfig struct {
	TimeoutSeconds             int `mapstructure:"timeout_seconds"`
	ReaperIntervalMilliseconds int `mapstructure:"reaper_interval_milliseconds"`
}

type LoggingConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	Tokens  TokensConfig  `mapstructure:"tokens"`
	Task    TaskConfig    `mapstructure:"task"`
	Logging LoggingConfig `mapstructure:"logging"`
}

var AppConfig *Config

func LoadConfig(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	err := v.ReadInConfig()
	if err != nil {
		return nil, err
	}
	var conf Config
	err = v.Unmarshal(&conf)
	if err != nil {
		return nil, err
	}
	return &conf, nil
}

func init() {
	conf, err := LoadConfig("./config.yaml")
	if err != nil {
		panic(err)
	}
	AppConfig = conf
}
