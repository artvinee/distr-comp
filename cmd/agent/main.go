package main

import (
	agent "distr-comp/internal/agent/client"
	"distr-comp/internal/logger"
	"os"
	"strconv"
)

func main() {
	logger.InitLogger(
		logger.Config{
			Level:      logger.InfoLevel,
			OutputPath: "stdout",
			Encoding:   "console",
		},
	)

	logger.Info("Starting agent service")

	computingPower := getEnvOrDefaultInt("COMPUTING_POWER", 10)
	logger.Infof("Computing power set to: %d", computingPower)

	orchestratorURL := getEnvOrDefault("ORCHESTRATOR_URL", "http://localhost:8080")
	logger.Infof("Orchestrator URL set to: %s", orchestratorURL)

	logger.Infof("Starting agent with %d computing goroutines", computingPower)
	agent.Start(computingPower, orchestratorURL)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		logger.Debugf("Environment variable %s found with value: %s", key, value)
		return value
	}
	logger.Debugf("Environment variable %s not found, using default: %s", key, defaultValue)
	return defaultValue
}

func getEnvOrDefaultInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			if intValue < 1 {
				logger.Warnf("Environment variable %s has invalid value: %s (less than 1), using default: %d", key, value, defaultValue)
				return defaultValue
			}
			return intValue
		} else {
			logger.Warnf("Failed to parse environment variable %s as integer: %s, using default: %d", key, value, defaultValue)
		}
	}
	logger.Debugf("Environment variable %s not found, using default: %d", key, defaultValue)
	return defaultValue
}
