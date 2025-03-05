package logger

type LogLevel string

type Config struct {
	Level      LogLevel
	OutputPath string
	Encoding   string
}
