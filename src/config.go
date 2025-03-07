package tango

import (
	"github.com/spf13/viper"
)

type ServerConfig struct {
	Name string `mapstructure:"name"`
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type TokensConfig struct {
	JWTSecret string `mapstructure:"JWTSecret"`
}

type TaskConfig struct {
	TimeoutSeconds             int `mapstructure:"timeout_seconds"`
	ReaperIntervalMilliseconds int `mapstructure:"reaper_interval_milliseconds"`
}

type LoggingConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

type GCPConfig struct {
	ProjectID           string `mapstructure:"project_id"`
	Location            string `mapstructure:"location"`
	WeightBucket        string `mapstructure:"weight_bucket"`
	RecordsBucket       string `mapstructure:"records_bucket"`
	KeyFile             string `mapstructure:"key_file"`
	JWTSecretName       string `mapstructure:"jwt_secret_name"`
	TestTokenSecretName string `mapstructure:"test_token_secret_name"`
	ServerCrt           string `mapstructure:"server_crt"`
	ServerKey           string `mapstructure:"server_key"`
}

type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	Tokens  TokensConfig  `mapstructure:"tokens"`
	Task    TaskConfig    `mapstructure:"task"`
	Logging LoggingConfig `mapstructure:"logging"`
	GCP     GCPConfig     `mapstructure:"gcp"`
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
