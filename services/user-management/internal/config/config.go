package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	UserService ServerConfig   `mapstructure:"user_service"`
	Database    DatabaseConfig `mapstructure:"database"`
	Redis       RedisConfig    `mapstructure:"redis"`
	Kafka       KafkaConfig    `mapstructure:"kafka"`
	JWT         JWTConfig      `mapstructure:"jwt"`
	Auth        AuthConfig     `mapstructure:"auth"`
	Services    ServicesConfig `mapstructure:"services"`

	// InternalKey guards /api/v1/internal (e.g. mail-service resolves a user's
	// email to send transactional mail). Empty closes those routes.
	InternalKey string `mapstructure:"internal_key"`
}

// ServicesConfig points at the other microservices user-management calls.
// payment-service owns subscriptions and gates store creation; mail-service
// sends the forgot-password email.
type ServicesConfig struct {
	PaymentServiceURL  string `mapstructure:"payment_service_url"`
	PaymentInternalKey string `mapstructure:"payment_internal_key"`
	MailServiceURL     string `mapstructure:"mail_service_url"`
	MailInternalKey    string `mapstructure:"mail_internal_key"`
}

// AuthConfig holds cookie behaviour and OAuth provider credentials.
type AuthConfig struct {
	FrontendURL  string              `mapstructure:"frontend_url"`  // where OAuth callbacks redirect back to
	CookieDomain string              `mapstructure:"cookie_domain"` // "" = host-only cookie
	CookieSecure bool                `mapstructure:"cookie_secure"` // true in production (HTTPS)
	Google       OAuthProviderConfig `mapstructure:"google"`
	Facebook     OAuthProviderConfig `mapstructure:"facebook"`
}

type OAuthProviderConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"` // must match the provider console exactly
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

type RedisConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
}

type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`
}

type JWTConfig struct {
	Secret     string `mapstructure:"secret"`
	Expires    string `mapstructure:"expires"`
	AccessTTL  string `mapstructure:"access_ttl"`
	RefreshTTL string `mapstructure:"refresh_ttl"`
}

func ConfigLoader() (*Config, error) {
	// ---------------------------------------------------------
	// 1. Determine application environment
	// ---------------------------------------------------------

	env := os.Getenv("APP_ENV")

	if env == "" {
		env = "dev"
		fmt.Println("⚠️ APP_ENV not set, defaulting to dev")
	}

	fmt.Println("Environment:", env)

	// ---------------------------------------------------------
	// 2. Base config.yaml
	// ---------------------------------------------------------

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf(
			"failed to read base config: %w",
			err,
		)
	}

	// ---------------------------------------------------------
	// 3. Environment config
	//    config.dev.yaml
	//    config.test.yaml
	//    config.prod.yaml
	// ---------------------------------------------------------

	viper.SetConfigName("config." + env)

	if err := viper.MergeInConfig(); err != nil {
		return nil, fmt.Errorf(
			"failed to read environment config %q: %w",
			env,
			err,
		)
	}

	// ---------------------------------------------------------
	// 4. Environment variables
	// ---------------------------------------------------------

	viper.SetEnvKeyReplacer(
		strings.NewReplacer(".", "_"),
	)

	viper.AutomaticEnv()

	if err := bindEnv(); err != nil {
		return nil, fmt.Errorf(
			"failed to bind environment variables: %w",
			err,
		)
	}

	// ---------------------------------------------------------
	// 5. Unmarshal
	// ---------------------------------------------------------

	var cfg Config

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal config: %w",
			err,
		)
	}

	fmt.Println("✅ Configuration loaded successfully")

	return &cfg, nil
}

func bindEnv() error {
	bindings := map[string]string{

		// -----------------------------------------------------
		// USER SERVICE
		// -----------------------------------------------------

		"user_service.host": "USER_SERVICE_HOST",
		"user_service.port": "USER_SERVICE_PORT",
		"user_service.mode": "USER_SERVICE_MODE",

		// -----------------------------------------------------
		// DATABASE
		// -----------------------------------------------------

		"database.host":     "USER_DB_HOST",
		"database.port":     "USER_DB_PORT",
		"database.user":     "POSTGRES_USER",
		"database.password": "POSTGRES_PASSWORD",
		"database.name":     "USER_DB_NAME",
		"database.dbssl":    "DATABASE_DBSSL",

		// -----------------------------------------------------
		// REDIS
		// -----------------------------------------------------

		"redis.host": "USER_REDIS_HOST",
		"redis.port": "USER_REDIS_PORT",

		// -----------------------------------------------------
		// KAFKA
		// -----------------------------------------------------

		"kafka.brokers": "USER_KAFKA_BROKERS",

		// -----------------------------------------------------
		// JWT
		// -----------------------------------------------------

		"jwt.secret":      "USER_JWT_SECRET",
		"jwt.expires":     "USER_JWT_EXPIRES",
		"jwt.access_ttl":  "USER_JWT_ACCESS_TTL",
		"jwt.refresh_ttl": "USER_JWT_REFRESH_TTL",

		// -----------------------------------------------------
		// AUTH / OAUTH
		// -----------------------------------------------------

		"auth.frontend_url":           "USER_AUTH_FRONTEND_URL",
		"auth.cookie_domain":          "USER_AUTH_COOKIE_DOMAIN",
		"auth.cookie_secure":          "USER_AUTH_COOKIE_SECURE",
		"auth.google.client_id":       "USER_AUTH_GOOGLE_CLIENT_ID",
		"auth.google.client_secret":   "USER_AUTH_GOOGLE_CLIENT_SECRET",
		"auth.google.redirect_url":    "USER_AUTH_GOOGLE_REDIRECT_URL",
		"auth.facebook.client_id":     "USER_AUTH_FACEBOOK_CLIENT_ID",
		"auth.facebook.client_secret": "USER_AUTH_FACEBOOK_CLIENT_SECRET",
		"auth.facebook.redirect_url":  "USER_AUTH_FACEBOOK_REDIRECT_URL",

		// -----------------------------------------------------
		// DOWNSTREAM SERVICES
		// -----------------------------------------------------

		"services.payment_service_url":  "USER_PAYMENT_SERVICE_URL",
		"services.payment_internal_key": "USER_PAYMENT_INTERNAL_KEY",
		"services.mail_service_url":     "USER_MAIL_SERVICE_URL",
		"services.mail_internal_key":    "USER_MAIL_INTERNAL_KEY",

		"internal_key": "USER_INTERNAL_KEY",
	}

	for key, env := range bindings {
		if err := viper.BindEnv(key, env); err != nil {
			return err
		}
	}

	return nil
}
