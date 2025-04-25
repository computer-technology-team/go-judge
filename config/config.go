package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	JudgeServer    ServerConfig         `mapstructure:"judge_server"`
	RunnerServer   ServerConfig         `mapstructure:"runner_server"`
	RunnerClient   ClientConfig         `mapstructure:"runner_client"`
	Database       DatabaseConfig       `mapstructure:"database"`
	Authentication AuthenticationConfig `mapstructure:"authentication"`
	Broker         BrokerConfig         `mapstructure:"broker"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type ClientConfig struct {
	Address string `mapstructure:"address"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxConns        int32         `mapstructure:"max_conns"`
	MinConns        int32         `mapstructure:"min_conns"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `mapstructure:"max_conn_idle_time"`
	ConnTimeout     time.Duration `mapstructure:"conn_timeout"`
}

type BrokerConfig struct {
	Workers    int           `mapstructure:"workers"`
	JobTimeout time.Duration `mapstructure:"job_timeout"`
}

// DSN returns a PostgreSQL connection string
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Username, c.Password, c.Host, c.Port, c.Name, c.SSLMode)
}

type AuthenticationConfig struct {
	Keys        map[string]string `mapstructure:"keys"`
	ActiveKeyID string            `mapstructure:"active_key_id"`
	TokenExpiry time.Duration     `mapstructure:"token_expiry"`
}

func Load(configPath string) (*Config, error) {
	v := viper.New()

	v.SetDefault("judge_server.port", 8080)
	v.SetDefault("judge_server.host", "0.0.0.0")
	v.SetDefault("runner_server.port", 8888)
	v.SetDefault("runner_server.host", "0.0.0.0")

	v.SetDefault("runner_client.address", "runner:8888")

	v.SetDefault("broker.workers", 5)
	v.SetDefault("broker.job_timeout", time.Minute*5)

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.username", "postgres")
	v.SetDefault("database.password", "postgres")
	v.SetDefault("database.database", "gojudge")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_conns", 10)
	v.SetDefault("database.min_conns", 2)
	v.SetDefault("database.max_conn_lifetime", 1*time.Hour)
	v.SetDefault("database.max_conn_idle_time", 30*time.Minute)
	v.SetDefault("database.conn_timeout", 5*time.Second)

	// Authentication defaults
	v.SetDefault("authentication.token_expiry", 24*time.Hour)
	// Default key is base64 encoded "default-secret-key-change-me-in-production"
	v.SetDefault("authentication.keys", map[string]string{
		"default": "ZGVmYXVsdC1zZWNyZXQta2V5LWNoYW5nZS1tZS1pbi1wcm9kdWN0aW9u",
	})
	v.SetDefault("authentication.active_key_id", "default")

	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// Config search paths
	if configPath != "" {
		v.AddConfigPath(configPath)
	}
	v.AddConfigPath("./configs") // look in configs directory
	v.AddConfigPath(".")         // look in working directory

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
	v.AutomaticEnv()

	// Read config
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, will use defaults and env vars
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	return &cfg, nil
}
