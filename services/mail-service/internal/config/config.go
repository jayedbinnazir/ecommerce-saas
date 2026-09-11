// Package config loads layered YAML config plus environment overrides.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Kafka    KafkaConfig    `mapstructure:"kafka"`
	SMTP     SMTPConfig     `mapstructure:"smtp"`
	Services ServicesConfig `mapstructure:"services"`

	// InternalKey guards every HTTP route (mail-service has no public API — only
	// other services send mail). Empty closes the API.
	InternalKey string `mapstructure:"internal_key"`
}

type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`
}

// ServicesConfig — mail-service resolves a recipient's name + email from
// user-management when it handles an order/payment event.
type ServicesConfig struct {
	UserServiceURL  string `mapstructure:"user_service_url"`
	UserInternalKey string `mapstructure:"user_internal_key"`
}

// SMTPConfig is the outbound mail relay. When Host is empty the sender runs in
// stub mode: it logs the message and marks it SENT without a real delivery.
type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"` // e.g. "Acme <no-reply@acme.test>"
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Host                  string        `mapstructure:"host"`
	Port                  string        `mapstructure:"port"`
	User                  string        `mapstructure:"user"`
	Password              string        `mapstructure:"password"`
	Name                  string        `mapstructure:"name"`
	DBSSL                 bool          `mapstructure:"dbssl"`
	MaxOpenConnections    int           `mapstructure:"max_open_connections"`
	MaxIdleConnections    int           `mapstructure:"max_idle_connections"`
	ConnectionMaxLifetime time.Duration `mapstructure:"connection_max_lifetime"`
}

func ConfigLoader() (*Config, error) {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
		fmt.Println("⚠️ APP_ENV not set, defaulting to dev")
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read base config: %w", err)
	}

	viper.SetConfigName("config." + env)
	if err := viper.MergeInConfig(); err != nil {
		return nil, fmt.Errorf("read %q config: %w", env, err)
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := bindEnv(); err != nil {
		return nil, fmt.Errorf("bind env: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}

func bindEnv() error {
	bindings := map[string]string{
		"server.host": "MAIL_SERVICE_HOST",
		"server.port": "MAIL_SERVICE_PORT",
		"server.mode": "MAIL_SERVICE_MODE",

		"database.host":     "MAIL_DB_HOST",
		"database.port":     "MAIL_DB_PORT",
		"database.user":     "POSTGRES_USER",
		"database.password": "POSTGRES_PASSWORD",
		"database.name":     "MAIL_DB_NAME",
		"database.dbssl":    "MAIL_DB_SSL",

		"kafka.brokers": "MAIL_KAFKA_BROKERS",

		"smtp.host":     "MAIL_SMTP_HOST",
		"smtp.port":     "MAIL_SMTP_PORT",
		"smtp.username": "MAIL_SMTP_USERNAME",
		"smtp.password": "MAIL_SMTP_PASSWORD",
		"smtp.from":     "MAIL_SMTP_FROM",

		"services.user_service_url":  "MAIL_USER_SERVICE_URL",
		"services.user_internal_key": "MAIL_USER_INTERNAL_KEY",

		"internal_key": "MAIL_INTERNAL_KEY",
	}

	for key, env := range bindings {
		if err := viper.BindEnv(key, env); err != nil {
			return err
		}
	}
	return nil
}
