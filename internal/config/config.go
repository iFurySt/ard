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

func AdminTokensFile(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv("ARD_ADMIN_TOKENS_FILE")
}

func PolicyFile(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv("ARD_POLICY_FILE")
}

func OTLPTracesEndpoint(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv("ARD_OTLP_TRACES_ENDPOINT")
}

func ConsoleDir(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv("ARD_CONSOLE_DIR")
}
