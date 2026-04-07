package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Log      LogConfig
	JWT      JWTConfig
	AI       AIConfigEnv
}

type AIConfigEnv struct {
	OpenAIAPIKey       string
	DefaultProvider    string
	DefaultModel       string
	DefaultMaxTokens   int
	DefaultTemperature float64
	DefaultDebounce    int
}

type ServerConfig struct {
	Port         string
	ReadTimeout  int
	WriteTimeout int
	SecureCookie bool
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type LogConfig struct {
	Level string
}

type JWTConfig struct {
	Secret     string
	Expiration time.Duration
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.SetDefault("server.port", "8533")
	viper.SetDefault("server.readtimeout", 10)
	viper.SetDefault("server.writetimeout", 10)
	viper.SetDefault("server.securecookie", false)
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", "3307")
	viper.SetDefault("database.user", "crm")
	viper.SetDefault("database.password", "crm_secret")
	viper.SetDefault("database.name", "crm_juridico")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("jwt.secret", "change-me-in-production")
	viper.SetDefault("jwt.expiration", "24h")
	viper.SetDefault("ai.defaultprovider", "openai")
	viper.SetDefault("ai.defaultmodel", "gpt-5.4-nano")
	viper.SetDefault("ai.defaultmaxtokens", 1024)
	viper.SetDefault("ai.defaulttemperature", 0.7)
	viper.SetDefault("ai.defaultdebounce", 8)

	_ = viper.ReadInConfig()

	cfg := &Config{
		Server: ServerConfig{
			Port:         viper.GetString("server.port"),
			ReadTimeout:  viper.GetInt("server.readtimeout"),
			WriteTimeout: viper.GetInt("server.writetimeout"),
			SecureCookie: viper.GetBool("server.securecookie"),
		},
		Database: DatabaseConfig{
			Host:     viper.GetString("database.host"),
			Port:     viper.GetString("database.port"),
			User:     viper.GetString("database.user"),
			Password: viper.GetString("database.password"),
			Name:     viper.GetString("database.name"),
		},
		Log: LogConfig{
			Level: viper.GetString("log.level"),
		},
		JWT: JWTConfig{
			Secret:     viper.GetString("jwt.secret"),
			Expiration: viper.GetDuration("jwt.expiration"),
		},
		AI: AIConfigEnv{
			OpenAIAPIKey:       viper.GetString("openai.apikey"),
			DefaultProvider:    viper.GetString("ai.defaultprovider"),
			DefaultModel:       viper.GetString("ai.defaultmodel"),
			DefaultMaxTokens:   viper.GetInt("ai.defaultmaxtokens"),
			DefaultTemperature: viper.GetFloat64("ai.defaulttemperature"),
			DefaultDebounce:    viper.GetInt("ai.defaultdebounce"),
		},
	}

	if cfg.Server.Port == "" {
		return nil, fmt.Errorf("server.port is required")
	}

	return cfg, nil
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		d.User, d.Password, d.Host, d.Port, d.Name,
	)
}
