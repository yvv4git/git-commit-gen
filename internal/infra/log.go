package infra

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewWithLogLevel(levelValue string) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(levelValue)
	if err != nil {
		return nil, fmt.Errorf("parse level: %w", err)
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(os.Stderr),
		zap.NewAtomicLevelAt(level),
	)

	return zap.New(core, zap.AddCaller()), nil
}
