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

func AdminToken(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv("ARD_ADMIN_TOKEN")
}

func PolicyFile(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv("ARD_POLICY_FILE")
}
