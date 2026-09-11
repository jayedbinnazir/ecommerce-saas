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
	JWT      JWTConfig      `mapstructure:"jwt"`
	Services ServicesConfig `mapstructure:"services"`

	// InternalKey guards POST /api/v1/internal/notifications (other services
	// create notifications). Empty closes that route.
	InternalKey string `mapstructure:"internal_key"`
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

type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
}

type ServicesConfig struct {
	UserServiceURL  string `mapstructure:"user_service_url"`
	MailServiceURL  string `mapstructure:"mail_service_url"`
	MailInternalKey string `mapstructure:"mail_internal_key"`
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
		"server.host": "NOTIFICATION_SERVICE_HOST",
		"server.port": "NOTIFICATION_SERVICE_PORT",
		"server.mode": "NOTIFICATION_SERVICE_MODE",

		"database.host":     "NOTIFICATION_DB_HOST",
		"database.port":     "NOTIFICATION_DB_PORT",
		"database.user":     "POSTGRES_USER",
		"database.password": "POSTGRES_PASSWORD",
		"database.name":     "NOTIFICATION_DB_NAME",
		"database.dbssl":    "NOTIFICATION_DB_SSL",

		"kafka.brokers": "NOTIFICATION_KAFKA_BROKERS",

		"jwt.secret": "NOTIFICATION_JWT_SECRET",

		"services.user_service_url":  "NOTIFICATION_USER_SERVICE_URL",
		"services.mail_service_url":  "NOTIFICATION_MAIL_SERVICE_URL",
		"services.mail_internal_key": "NOTIFICATION_MAIL_INTERNAL_KEY",

		"internal_key": "NOTIFICATION_INTERNAL_KEY",
	}

	for key, env := range bindings {
		if err := viper.BindEnv(key, env); err != nil {
			return err
		}
	}
	return nil
}
