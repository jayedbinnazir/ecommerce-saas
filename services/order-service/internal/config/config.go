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

// JWTConfig only needs the shared secret: order-service verifies the access token
// issued by user-management, it never mints one.
type JWTConfig struct {
	Secret string `mapstructure:"secret"`
}

// ServicesConfig holds what order-service needs to reach its dependencies:
//   - user-management for the shared auth guard,
//   - cart-service to read the shopper's cart at checkout,
//   - inventory-service (bulk internal API + key) to reserve/release/ship stock.
type ServicesConfig struct {
	UserServiceURL       string `mapstructure:"user_service_url"`
	CartServiceURL       string `mapstructure:"cart_service_url"`
	InventoryServiceURL  string `mapstructure:"inventory_service_url"`
	InventoryInternalKey string `mapstructure:"inventory_internal_key"`
	PaymentServiceURL    string `mapstructure:"payment_service_url"`
	PaymentInternalKey   string `mapstructure:"payment_internal_key"`
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
		"server.host": "ORDER_SERVICE_HOST",
		"server.port": "ORDER_SERVICE_PORT",
		"server.mode": "ORDER_SERVICE_MODE",

		"database.host":     "ORDER_DB_HOST",
		"database.port":     "ORDER_DB_PORT",
		"database.user":     "POSTGRES_USER",
		"database.password": "POSTGRES_PASSWORD",
		"database.name":     "ORDER_DB_NAME",
		"database.dbssl":    "ORDER_DB_SSL",

		"kafka.brokers": "ORDER_KAFKA_BROKERS",

		"jwt.secret": "ORDER_JWT_SECRET",

		"services.user_service_url":       "ORDER_USER_SERVICE_URL",
		"services.cart_service_url":       "ORDER_CART_SERVICE_URL",
		"services.inventory_service_url":  "ORDER_INVENTORY_SERVICE_URL",
		"services.inventory_internal_key": "ORDER_INVENTORY_INTERNAL_KEY",
		"services.payment_service_url":    "ORDER_PAYMENT_SERVICE_URL",
		"services.payment_internal_key":   "ORDER_PAYMENT_INTERNAL_KEY",
	}

	for key, env := range bindings {
		if err := viper.BindEnv(key, env); err != nil {
			return err
		}
	}
	return nil
}
