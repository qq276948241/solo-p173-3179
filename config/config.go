package config

import "os"

type Config struct {
	DBPath     string
	JWTSecret  string
	ServerPort string
}

func Load() *Config {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "bnb.db"
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "bnb-secret-key-change-in-production"
	}
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080"
	}
	return &Config{
		DBPath:     dbPath,
		JWTSecret:  jwtSecret,
		ServerPort: serverPort,
	}
}
