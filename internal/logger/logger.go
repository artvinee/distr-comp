package logger

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LoggerMap struct {
	loggers map[string]*zap.Logger
	mu      sync.RWMutex
}

var (
	log       *zap.Logger
	loggerMap *LoggerMap
	once      sync.Once
)

const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
)

func InitLogger(cfg Config) {
	once.Do(func() {
		loggerMap = &LoggerMap{
			loggers: make(map[string]*zap.Logger),
		}

		log = createLogger(cfg)

		loggerMap.mu.Lock()
		loggerMap.loggers["default"] = log
		loggerMap.mu.Unlock()
	})
}

func createLogger(cfg Config) *zap.Logger {
	var level zapcore.Level
	switch cfg.Level {
	case DebugLevel:
		level = zapcore.DebugLevel
	case InfoLevel:
		level = zapcore.InfoLevel
	case WarnLevel:
		level = zapcore.WarnLevel
	case ErrorLevel:
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	var output zapcore.WriteSyncer
	if cfg.OutputPath == "stdout" {
		output = zapcore.AddSync(os.Stdout)
	} else if cfg.OutputPath == "stderr" {
		output = zapcore.AddSync(os.Stderr)
	} else {
		file, err := os.OpenFile(cfg.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			output = zapcore.AddSync(os.Stdout)
		} else {
			output = zapcore.AddSync(file)
		}
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	if cfg.Encoding == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	core := zapcore.NewCore(encoder, output, level)
	return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
}

func GetLogger() *zap.Logger {
	if log == nil {
		InitLogger(Config{
			Level:      InfoLevel,
			OutputPath: "stdout",
			Encoding:   "console",
		})
	}
	return log
}

func GetLoggerMap() *LoggerMap {
	if loggerMap == nil {
		InitLogger(Config{
			Level:      InfoLevel,
			OutputPath: "stdout",
			Encoding:   "console",
		})
	}
	return loggerMap
}

func GetNamedLogger(name string) *zap.Logger {
	loggerMap := GetLoggerMap()

	loggerMap.mu.RLock()
	logger, exists := loggerMap.loggers[name]
	loggerMap.mu.RUnlock()

	if exists {
		return logger
	}

	newLogger := createLogger(Config{
		Level:      InfoLevel,
		OutputPath: "stdout",
		Encoding:   "console",
	})

	loggerMap.mu.Lock()
	loggerMap.loggers[name] = newLogger
	loggerMap.mu.Unlock()

	return newLogger
}

func AddLogger(name string, cfg Config) *zap.Logger {
	loggerMap := GetLoggerMap()

	newLogger := createLogger(cfg)

	loggerMap.mu.Lock()
	loggerMap.loggers[name] = newLogger
	loggerMap.mu.Unlock()

	return newLogger
}

func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}

func Debugf(template string, args ...interface{}) {
	GetLogger().Sugar().Debugf(template, args...)
}

func Infof(template string, args ...interface{}) {
	GetLogger().Sugar().Infof(template, args...)
}

func Warnf(template string, args ...interface{}) {
	GetLogger().Sugar().Warnf(template, args...)
}

func Errorf(template string, args ...interface{}) {
	GetLogger().Sugar().Errorf(template, args...)
}

func Fatalf(template string, args ...interface{}) {
	GetLogger().Sugar().Fatalf(template, args...)
}
