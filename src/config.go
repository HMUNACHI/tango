/*
Tango is a product of Cactus Compute, Inc.
This code is proprietary. Do not share the code.
*/
package tango

import (
	"github.com/spf13/viper"
)

// ServerConfig holds the configuration for the Tango server, including its name, host, and port.
type ServerConfig struct {
	Name string `mapstructure:"name"`
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// TokensConfig holds configuration details for JWT secrets used by the Tango service.
type TokensConfig struct {
	JWTSecret string `mapstructure:"JWTSecret"`
}

// TaskConfig holds configuration parameters for task processing, such as timeout and reaper interval.
type TaskConfig struct {
	TimeoutSeconds             int `mapstructure:"timeout_seconds"`
	ReaperIntervalMilliseconds int `mapstructure:"reaper_interval_milliseconds"`
}

// LoggingConfig holds configuration details for logging, including log level and file path.
type LoggingConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

// GCPConfig holds configuration settings required to connect to Google Cloud Platform services,
// including project ID, location, bucket names, key file, secret names, and server certificates.
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

// Config aggregates all configuration settings for the Tango application.
type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	Tokens  TokensConfig  `mapstructure:"tokens"`
	Task    TaskConfig    `mapstructure:"task"`
	Logging LoggingConfig `mapstructure:"logging"`
	GCP     GCPConfig     `mapstructure:"gcp"`
}

// AppConfig is the global configuration for the Tango application.
var AppConfig *Config

// LoadConfig reads configuration from the specified file path using Viper,
// unmarshals it into a Config struct, and returns a pointer to it.
// It returns an error if reading or unmarshalling the configuration fails.
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

// init is called automatically to load the application configuration from "./config.yaml"
// and assigns it to the global variable AppConfig.
func init() {
	conf, err := LoadConfig("./config.yaml")
	if err != nil {
		panic(err)
	}
	AppConfig = conf
}
