package config

import "os"

const DefaultDatabaseURL = "postgres://postgres:postgres@localhost:5432/ard?sslmode=disable"

func DatabaseURL(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if value := os.Getenv("DATABASE_URL"); value != "" {
		return value
	}
	return DefaultDatabaseURL
}
