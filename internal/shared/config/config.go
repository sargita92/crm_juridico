package config

import (
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Log      LogConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  int
	WriteTimeout int
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
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", "3307")
	viper.SetDefault("database.user", "crm")
	viper.SetDefault("database.password", "crm_secret")
	viper.SetDefault("database.name", "crm_juridico")
	viper.SetDefault("log.level", "info")

	_ = viper.ReadInConfig()

	cfg := &Config{
		Server: ServerConfig{
			Port:         viper.GetString("server.port"),
			ReadTimeout:  viper.GetInt("server.readtimeout"),
			WriteTimeout: viper.GetInt("server.writetimeout"),
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
