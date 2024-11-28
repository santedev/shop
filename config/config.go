package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Production              bool
	PublicHost              string
	Port                    string
	DBPort                  string
	DBUser                  string
	DBHost                  string
	DBPassword              string
	DBAddress               string
	DBName                  string
	DBsslMode               string
	CookiesPath             string
	CookiesAuthSecret       string
	CookiesAuthAgeInSeconds int
	CookiesAuthIsSecure     bool
	CookiesAuthIsHttpOnly   bool
	GoogleKey               string
	GoogleSecret            string
	PaypalKey               string
	PaypalSecret            string
}

const (
	twoDaysInSeconds = 60 * 60 * 24 * 2
)

var Envs Config

func InitEnvConfig() Config {
	return Config{
		Production:              getEnvAsBool("PROD", false),
		PublicHost:              getEnv("HOST", "http://localhost"),
		Port:                    getEnv("PORT", "8000"),
		DBPort:                  getEnv("DB_PORT", "5432"),
		DBHost:                  getEnv("DB_HOST", "localhost"),
		DBUser:                  getEnv("DB_USER", "postgres"),
		DBPassword:              getEnv("DB_PASSWORD", "a$$word"),
		DBAddress:               fmt.Sprintf("%s:%s", getEnv("DB_HOST", "127.0.0.1"), getEnv("DB_PORT", "5432")),
		DBName:                  getEnv("DB_NAME", "postgres"),
		DBsslMode:               getEnv("DB_SSL_MODE", "require"),
		CookiesPath:             getEnv("COOKIES_AUTH_PATH", "/"),
		CookiesAuthSecret:       getEnv("COOKIES_AUTH_SECRET", "secret_cookie"),
		CookiesAuthAgeInSeconds: getEnvAsInt("COOKIES_AUTH_AGE", twoDaysInSeconds),
		CookiesAuthIsSecure:     getEnvAsBool("COOKIES_AUTH_IS_SECURE", true),
		CookiesAuthIsHttpOnly:   getEnvAsBool("COOKIES_AUTH_IS_HTTP_ONLY", true),
		GoogleKey:               getEnvOrError("GOOGLE_KEY"),
		GoogleSecret:            getEnvOrError("GOOGLE_SECRET"),
		PaypalKey:               getEnvOrError("PAYPAL_KEY"),
		PaypalSecret:            getEnvOrError("PAYPAL_SECRET"),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) > 0 {
		return value
	}

	return fallback
}

func getEnvOrError(key string) string {
	value := os.Getenv(key)
	if len(value) > 0 {
		return value
	}

	panic(fmt.Sprintf("Environment variable %s is not set", key))
}

func getEnvAsInt(key string, fallback int) int {
	value := os.Getenv(key)
	i, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return i
}

func getEnvAsBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	b, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return b
}

func LoadEnv() error {
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}
	root, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	for {
		err := godotenv.Load(filepath.Join(currentDir, ".env"))
		if err == nil {
			Envs = InitEnvConfig()
			return nil
		}
		if _, err := os.Stat(filepath.Join(currentDir, "main.go")); err == nil {
			return fmt.Errorf("couldnt find .env file, found main.go before file .env")
		}
		currentDir = filepath.Dir(currentDir)
		if currentDir == root {
			return fmt.Errorf("couldnt find .env file, found home dir before file .env")
		}
	}
}
