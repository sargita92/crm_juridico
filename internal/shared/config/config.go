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
	Files    FilesConfig
	Billing  BillingConfig
	Env      string // "development", "test", "production"
}

type BillingConfig struct {
	Cron      string // cron spec (default "0 3 * * *")
	GraceDays int    // dias uteis de carencia antes de marcar atrasado
	Timezone  string // IANA tz (default "America/Sao_Paulo")
}

type FilesConfig struct {
	StorageRoot string // where LocalDiskStorage persists captured media
	MaxBytes    int64  // per-file ceiling; 0 = 50 MB default
}

type AIConfigEnv struct {
	OpenAIAPIKey            string
	DefaultProvider         string
	DefaultModel            string
	DefaultMaxTokens        int
	DefaultTemperature      float64
	DefaultDebounce         int
	PlaygroundEnabled       bool
	ResetCommandEnabled     bool
	ToolLoopMaxIterations   int
	ToolCallMaxPerIteration int
	ToolExecutionTimeout    int
	ToolResultMaxLength     int
}

type ServerConfig struct {
	Port         string
	ReadTimeout  int
	WriteTimeout int
	SecureCookie bool
	// PprofEnabled habilita /debug/pprof/* (atrás de auth+admin). Default false;
	// ligar só sob demanda para investigação (env PPROF_ENABLED).
	PprofEnabled bool
	// SlowRequestThresholdMs: requests acima deste tempo (ms) são logados em Warn
	// ("slow http request"). 0 desativa. Header X-Response-Time é sempre adicionado.
	SlowRequestThresholdMs int
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	// SlowQueryThresholdMs: queries que demoram mais que isto (ms) são logadas
	// em Warn pelo logger do Gorm. 0 desativa o slow-query log.
	SlowQueryThresholdMs int
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
	viper.SetDefault("server.pprofenabled", false)
	_ = viper.BindEnv("server.pprofenabled", "PPROF_ENABLED")
	viper.SetDefault("server.slowrequestthresholdms", 1000)
	_ = viper.BindEnv("server.slowrequestthresholdms", "HTTP_SLOW_REQUEST_THRESHOLD_MS")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", "3307")
	viper.SetDefault("database.user", "crm")
	viper.SetDefault("database.password", "crm_secret")
	viper.SetDefault("database.name", "crm_juridico")
	viper.SetDefault("database.slowquerythresholdms", 200)
	_ = viper.BindEnv("database.slowquerythresholdms", "DB_SLOW_QUERY_THRESHOLD_MS")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("jwt.secret", "change-me-in-production")
	viper.SetDefault("jwt.expiration", "24h")
	viper.SetDefault("ai.defaultprovider", "openai")
	viper.SetDefault("ai.defaultmodel", "gpt-5.4-nano")
	viper.SetDefault("ai.defaultmaxtokens", 1024)
	viper.SetDefault("ai.defaulttemperature", 0.7)
	viper.SetDefault("ai.defaultdebounce", 8)
	viper.SetDefault("env", "development")
	viper.SetDefault("ai.playgroundenabled", false)
	viper.SetDefault("ai.resetcommandenabled", true)
	viper.SetDefault("ai.tool_loop_max_iterations", 5)
	viper.SetDefault("ai.tool_call_max_per_iteration", 10)
	viper.SetDefault("ai.tool_execution_timeout", 10)
	viper.SetDefault("ai.tool_result_max_length", 4000)
	viper.SetDefault("files.storageroot", "storage/files")
	viper.SetDefault("files.maxbytes", int64(50*1024*1024))
	_ = viper.BindEnv("files.storageroot", "FILES_STORAGE_ROOT")
	_ = viper.BindEnv("files.maxbytes", "FILES_MAX_BYTES")

	viper.SetDefault("billing.cron", "0 3 * * *")
	viper.SetDefault("billing.gracedays", 1)
	viper.SetDefault("billing.timezone", "America/Sao_Paulo")
	_ = viper.BindEnv("billing.cron", "BILLING_CRON")
	_ = viper.BindEnv("billing.gracedays", "BILLING_GRACE_DAYS")
	_ = viper.BindEnv("billing.timezone", "BILLING_TZ")

	_ = viper.BindEnv("env", "ENV")
	_ = viper.BindEnv("openai.apikey", "OPENAI_API_KEY")
	_ = viper.BindEnv("ai.playgroundenabled", "AI_PLAYGROUND_ENABLED")
	_ = viper.BindEnv("ai.resetcommandenabled", "AI_RESET_COMMAND_ENABLED")
	_ = viper.BindEnv("ai.tool_loop_max_iterations", "AI_TOOL_LOOP_MAX_ITERATIONS")
	_ = viper.BindEnv("ai.tool_call_max_per_iteration", "AI_TOOL_CALL_MAX_PER_ITERATION")
	_ = viper.BindEnv("ai.tool_execution_timeout", "AI_TOOL_EXECUTION_TIMEOUT_SECONDS")
	_ = viper.BindEnv("ai.tool_result_max_length", "AI_TOOL_RESULT_MAX_LENGTH")

	_ = viper.ReadInConfig()

	cfg := &Config{
		Server: ServerConfig{
			Port:                   viper.GetString("server.port"),
			ReadTimeout:            viper.GetInt("server.readtimeout"),
			WriteTimeout:           viper.GetInt("server.writetimeout"),
			SecureCookie:           viper.GetBool("server.securecookie"),
			PprofEnabled:           viper.GetBool("server.pprofenabled"),
			SlowRequestThresholdMs: viper.GetInt("server.slowrequestthresholdms"),
		},
		Database: DatabaseConfig{
			Host:                 viper.GetString("database.host"),
			Port:                 viper.GetString("database.port"),
			User:                 viper.GetString("database.user"),
			Password:             viper.GetString("database.password"),
			Name:                 viper.GetString("database.name"),
			SlowQueryThresholdMs: viper.GetInt("database.slowquerythresholdms"),
		},
		Log: LogConfig{
			Level: viper.GetString("log.level"),
		},
		JWT: JWTConfig{
			Secret:     viper.GetString("jwt.secret"),
			Expiration: viper.GetDuration("jwt.expiration"),
		},
		AI: AIConfigEnv{
			OpenAIAPIKey:            viper.GetString("openai.apikey"),
			DefaultProvider:         viper.GetString("ai.defaultprovider"),
			DefaultModel:            viper.GetString("ai.defaultmodel"),
			DefaultMaxTokens:        viper.GetInt("ai.defaultmaxtokens"),
			DefaultTemperature:      viper.GetFloat64("ai.defaulttemperature"),
			DefaultDebounce:         viper.GetInt("ai.defaultdebounce"),
			PlaygroundEnabled:       viper.GetBool("ai.playgroundenabled"),
			ResetCommandEnabled:     viper.GetBool("ai.resetcommandenabled"),
			ToolLoopMaxIterations:   viper.GetInt("ai.tool_loop_max_iterations"),
			ToolCallMaxPerIteration: viper.GetInt("ai.tool_call_max_per_iteration"),
			ToolExecutionTimeout:    viper.GetInt("ai.tool_execution_timeout"),
			ToolResultMaxLength:     viper.GetInt("ai.tool_result_max_length"),
		},
		Files: FilesConfig{
			StorageRoot: viper.GetString("files.storageroot"),
			MaxBytes:    viper.GetInt64("files.maxbytes"),
		},
		Billing: BillingConfig{
			Cron:      viper.GetString("billing.cron"),
			GraceDays: viper.GetInt("billing.gracedays"),
			Timezone:  viper.GetString("billing.timezone"),
		},
		Env: viper.GetString("env"),
	}

	if cfg.Server.Port == "" {
		return nil, fmt.Errorf("server.port is required")
	}

	return cfg, nil
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true",
		d.User, d.Password, d.Host, d.Port, d.Name,
	)
}
