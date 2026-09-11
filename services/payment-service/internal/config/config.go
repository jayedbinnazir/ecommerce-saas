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
	Stripe   StripeConfig   `mapstructure:"stripe"`

	// InternalKey guards the service-to-service endpoints under /api/v1/internal
	// (order-service drives payments, user-management checks subscriptions).
	InternalKey string `mapstructure:"internal_key"`
}

// StripeConfig holds the card-payment provider credentials. When SecretKey is
// empty the gateway runs in stub mode (card payments "succeed" instantly).
type StripeConfig struct {
	SecretKey     string `mapstructure:"secret_key"`     // sk_test_... / sk_live_...
	WebhookSecret string `mapstructure:"webhook_secret"` // whsec_... — enables POST /webhooks/stripe
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

// JWTConfig only needs the shared secret: payment-service verifies the access
// token issued by user-management, it never mints one.
type JWTConfig struct {
	Secret string `mapstructure:"secret"`
}

type ServicesConfig struct {
	// UserServiceURL is user-management's base URL, used to resolve tenant role
	// when a manager reads another customer's payment.
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
		"server.host": "PAYMENT_SERVICE_HOST",
		"server.port": "PAYMENT_SERVICE_PORT",
		"server.mode": "PAYMENT_SERVICE_MODE",

		"database.host":     "PAYMENT_DB_HOST",
		"database.port":     "PAYMENT_DB_PORT",
		"database.user":     "POSTGRES_USER",
		"database.password": "POSTGRES_PASSWORD",
		"database.name":     "PAYMENT_DB_NAME",
		"database.dbssl":    "PAYMENT_DB_SSL",

		"kafka.brokers": "PAYMENT_KAFKA_BROKERS",

		"jwt.secret": "PAYMENT_JWT_SECRET",

		"services.user_service_url": "PAYMENT_USER_SERVICE_URL",

		"stripe.secret_key":     "PAYMENT_STRIPE_SECRET_KEY",
		"stripe.webhook_secret": "PAYMENT_STRIPE_WEBHOOK_SECRET",

		"internal_key": "PAYMENT_INTERNAL_KEY",
	}

	for key, env := range bindings {
		if err := viper.BindEnv(key, env); err != nil {
			return err
		}
	}
	return nil
}
