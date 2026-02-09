package utils

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// InitLogger configures a Zap logger that writes to logs/deployment.log.
// It is critical that we do NOT log to stdout/stderr while the TUI is running.
func InitLogger() (*zap.Logger, error) {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Ensure logs directory exists
	if err := os.MkdirAll("logs", 0755); err != nil {
		return nil, err
	}

	// Open the log file
	file, err := os.OpenFile("logs/deployment.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	// Create a core that writes ONLY to the file
	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(file),
		zap.InfoLevel,
	)

	logger := zap.New(fileCore)
	return logger, nil
}
