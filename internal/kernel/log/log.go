package log

import (
	"io"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	FormatJSON    = "json"
	FormatConsole = "console"

	AppFile   = "app.log"
	ErrorFile = "errors.log"

	defaultMaxSizeMB  = 20
	defaultMaxBackups = 5
	defaultMaxAgeDays = 30
	directoryMode     = 0o750
)

type FileConfig struct {
	Dir        string
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Enabled    bool
	Compress   bool
}

type Config struct {
	File    *FileConfig
	Level   string
	Format  string
	Service string
	Version string
	Console bool
}

type Logger struct {
	*zap.Logger
	Level zap.AtomicLevel
	sinks []io.Closer
}

func (l *Logger) Close() error {
	var failure error
	for _, sink := range l.sinks {
		if err := sink.Close(); err != nil && failure == nil {
			failure = errors.Wrap(err, errors.Internal, "close log sink")
		}
	}
	return failure
}

func New(cfg Config) (*Logger, error) {
	level := zap.NewAtomicLevelAt(parseLevel(cfg.Level))

	var cores []zapcore.Core
	var sinks []io.Closer

	if cfg.Console {
		cores = append(cores, redacting{Core: zapcore.NewCore(consoleEncoder(cfg.Format), zapcore.AddSync(os.Stdout), level)})
	}

	if cfg.File != nil && cfg.File.Enabled {
		fileCores, files, err := fileCores(cfg.File, level)
		if err != nil {
			return nil, err
		}
		cores = append(cores, fileCores...)
		sinks = files
	}

	logger := zap.New(zapcore.NewTee(cores...), zap.AddCaller()).
		With(zap.String("service", cfg.Service), zap.String("version", cfg.Version))

	return &Logger{Logger: logger, Level: level, sinks: sinks}, nil
}

func fileCores(cfg *FileConfig, level zapcore.LevelEnabler) ([]zapcore.Core, []io.Closer, error) {
	if err := os.MkdirAll(cfg.Dir, directoryMode); err != nil {
		return nil, nil, errors.Wrap(err, errors.Internal, "create log directory")
	}

	encoder := zapcore.NewJSONEncoder(fileEncoderConfig())
	app := rotatingSink(cfg, AppFile)
	failures := rotatingSink(cfg, ErrorFile)

	return []zapcore.Core{
			redacting{Core: zapcore.NewCore(encoder, zapcore.AddSync(app), level)},
			redacting{Core: zapcore.NewCore(encoder, zapcore.AddSync(failures), zapcore.ErrorLevel)},
		},
		[]io.Closer{app, failures},
		nil
}

func rotatingSink(cfg *FileConfig, name string) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   filepath.Join(cfg.Dir, name),
		MaxSize:    orDefault(cfg.MaxSizeMB, defaultMaxSizeMB),
		MaxBackups: orDefault(cfg.MaxBackups, defaultMaxBackups),
		MaxAge:     orDefault(cfg.MaxAgeDays, defaultMaxAgeDays),
		Compress:   cfg.Compress,
	}
}

func orDefault(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func consoleEncoder(format string) zapcore.Encoder {
	if format == FormatConsole {
		cfg := zap.NewDevelopmentEncoderConfig()
		cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.EncodeTime = zapcore.ISO8601TimeEncoder
		cfg.EncodeDuration = zapcore.StringDurationEncoder
		cfg.EncodeCaller = zapcore.ShortCallerEncoder
		return zapcore.NewConsoleEncoder(cfg)
	}
	return zapcore.NewJSONEncoder(fileEncoderConfig())
}

func fileEncoderConfig() zapcore.EncoderConfig {
	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeLevel = zapcore.LowercaseLevelEncoder
	cfg.TimeKey = "t"
	cfg.LevelKey = "l"
	cfg.NameKey = "logger"
	cfg.CallerKey = "c"
	cfg.MessageKey = "m"
	cfg.StacktraceKey = "st"
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncodeDuration = zapcore.StringDurationEncoder
	cfg.EncodeCaller = zapcore.ShortCallerEncoder
	return cfg
}

func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}
