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
	S3       S3Config       `mapstructure:"s3"`
}

// S3Config points at the object store used for product images. Endpoint is set
// for MinIO/LocalStack in dev and left empty for real AWS. PublicBaseURL is the
// CDN (or bucket) URL images are served from.
type S3Config struct {
	Bucket          string `mapstructure:"bucket"`
	Region          string `mapstructure:"region"`
	Endpoint        string `mapstructure:"endpoint"`
	PublicBaseURL   string `mapstructure:"public_base_url"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
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

// JWTConfig only needs the shared secret: product-service verifies the access
// token issued by user-management, it never mints one.
type JWTConfig struct {
	Secret string `mapstructure:"secret"`
}

type ServicesConfig struct {
	// UserServiceURL is the base URL of user-management (no trailing slash),
	// e.g. http://user-management:8080
	UserServiceURL string `mapstructure:"user_service_url"`
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
		"server.host": "PRODUCT_SERVICE_HOST",
		"server.port": "PRODUCT_SERVICE_PORT",
		"server.mode": "PRODUCT_SERVICE_MODE",

		"database.host":     "PRODUCT_DB_HOST",
		"database.port":     "PRODUCT_DB_PORT",
		"database.user":     "POSTGRES_USER",
		"database.password": "POSTGRES_PASSWORD",
		"database.name":     "PRODUCT_DB_NAME",
		"database.dbssl":    "PRODUCT_DB_SSL",

		"kafka.brokers": "PRODUCT_KAFKA_BROKERS",

		"jwt.secret": "PRODUCT_JWT_SECRET",

		"services.user_service_url": "PRODUCT_USER_SERVICE_URL",

		"s3.bucket":            "PRODUCT_S3_BUCKET",
		"s3.region":            "PRODUCT_S3_REGION",
		"s3.endpoint":          "PRODUCT_S3_ENDPOINT",
		"s3.public_base_url":   "PRODUCT_S3_PUBLIC_BASE_URL",
		"s3.access_key_id":     "PRODUCT_S3_ACCESS_KEY_ID",
		"s3.secret_access_key": "PRODUCT_S3_SECRET_ACCESS_KEY",
	}

	for key, env := range bindings {
		if err := viper.BindEnv(key, env); err != nil {
			return err
		}
	}
	return nil
}
