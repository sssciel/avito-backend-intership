package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

var envFiles = []string{
	"./cfg/.env",
}

var envVars = []string{
	"DB_HOST",
	"DB_PORT",
	"DB_USER",
	"DB_PASSWORD",
	"DB_NAME",

	"SERVICE_API_TOKEN",
	"SERVICE_PORT",
}

type DBConfig struct {
	Host     string `mapstructure:"DB_HOST"`
	Port     string `mapstructure:"DB_PORT"`
	User     string `mapstructure:"DB_USER"`
	Password string `mapstructure:"DB_PASSWORD"`
	DBName   string `mapstructure:"DB_NAME"`
}

type ServiceConfig struct {
	Port  string `mapstructure:"SERVICE_HOST"`
	Token string `mapstructure:"SERVICE_API_TOKEN"`
}

var (
	DB  DBConfig
	API ServiceConfig
)

var ConfigStructs = []interface{}{
	&DB,
	&API,
}

func loadEnv() {
	slog.Debug("Loading .env files")
	// Попытка загрузить .env файл, но не критично если его нет (для Docker)
	err := godotenv.Load(envFiles...)
	if err != nil {
		slog.Warn(".env file not found, using environment variables", "err", err)
	}
	viper.AutomaticEnv()

	for _, v := range envVars {
		viper.BindEnv(v)
	}
}

func loadStructs() {
	slog.Debug("Loading config structs")
	for _, v := range ConfigStructs {
		err := viper.Unmarshal(v)
		if err != nil {
			slog.Error("Error unmarshaling config struct", "err", err)
			os.Exit(1)
		}
	}
}

func SetupConfigs() {
	loadEnv()
	loadStructs()
}

func GetDBURL(driver string) string {
	sslmode := os.Getenv("DB_SSLMODE")
	if sslmode == "" {
		sslmode = "require"
	}

	switch driver {
	case "postgresql":
		return fmt.Sprintf(
			"postgresql://%s:%s@%s:%s/%s?sslmode=%s",
			DB.User,
			DB.Password,
			DB.Host,
			DB.Port,
			DB.DBName,
			sslmode,
		)
	default:
		return ""
	}
}
