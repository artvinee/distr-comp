package main

import (
	"distr-comp/internal/logger"
	server "distr-comp/internal/orchestrator/server"
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {
	timeAddition := parseDurationEnv("TIME_ADDITION", 2000)
	timeSubtraction := parseDurationEnv("TIME_SUBTRACTION", 2000)
	timeMultiplication := parseDurationEnv("TIME_MULTIPLICATION", 2000)
	timeDivision := parseDurationEnv("TIME_DIVISION", 2000)
	port := getEnvOrDefaultInt("PORT", 8080)

	server := server.NewServer(timeAddition, timeSubtraction, timeMultiplication, timeDivision)
	server.Run(fmt.Sprintf(":%d", port))
}

func parseDurationEnv(key string, defaultValue int) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			if intValue < 1 {
				return time.Duration(defaultValue)
			}
			return time.Duration(intValue)
		}
	}
	return time.Duration(defaultValue)
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
