package logger

import (
	"Postulator/internal/config"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

var (
	globalLogger *Logger
	once         sync.Once
)

type Logger struct {
	zlog       zerolog.Logger
	scope      string
	appFile    *os.File
	errFile    *os.File
	logDir     string
	appLogPath string
	errLogPath string
}

type LogReader struct {
	filePath string
}

type LogEntry struct {
	Level     string         `json:"level"`
	Timestamp string         `json:"time"`
	Scope     string         `json:"scope"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"-"`
}

func New(cfg *config.Config) (*Logger, error) {
	var err error
	once.Do(func() {
		globalLogger, err = createLogger(cfg)
	})

	if err != nil {
		return nil, err
	}

	return globalLogger, nil
}

func NewForTest(cfg *config.Config) (*Logger, error) {
	return createLogger(cfg)
}

func createLogger(cfg *config.Config) (*Logger, error) {
	zerolog.TimeFieldFormat = time.RFC3339

	if cfg.LogDir == "" {
		cfg.LogDir = "logs"
	}
	if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	if cfg.AppLogFile == "" {
		cfg.AppLogFile = "app.log"
	}
	if cfg.ErrLogFile == "" {
		cfg.ErrLogFile = "errors.log"
	}

	appLogPath := filepath.Join(cfg.LogDir, cfg.AppLogFile)
	appFile, err := os.OpenFile(appLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open app log file: %w", err)
	}

	errLogPath := filepath.Join(cfg.LogDir, cfg.ErrLogFile)
	errFile, err := os.OpenFile(errLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		_ = appFile.Close()
		return nil, fmt.Errorf("failed to open error log file: %w", err)
	}

	errWriter := &levelWriter{
		writer:   errFile,
		minLevel: zerolog.ErrorLevel,
	}

	var writers []io.Writer
	writers = append(writers, appFile)
	writers = append(writers, errWriter)

	if cfg.ConsoleOut {
		if cfg.PrettyPrint {
			consoleWriter := zerolog.ConsoleWriter{
				Out:        os.Stdout,
				TimeFormat: "15:04:05",
				NoColor:    false,
			}
			writers = append(writers, consoleWriter)
		} else {
			writers = append(writers, os.Stdout)
		}
	}

	multiWriter := io.MultiWriter(writers...)

	level := parseLevel(cfg.LogLevel)
	zerolog.SetGlobalLevel(level)

	zlog := zerolog.New(multiWriter).With().Timestamp().Logger()

	return &Logger{
		zlog:       zlog,
		scope:      "app",
		appFile:    appFile,
		errFile:    errFile,
		logDir:     cfg.LogDir,
		appLogPath: appLogPath,
		errLogPath: errLogPath,
	}, nil
}

func parseLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

func (l *Logger) Close() error {
	var errs []error

	if l.appFile != nil {
		if err := l.appFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("app log: %w", err))
		}
	}

	if l.errFile != nil {
		if err := l.errFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("error log: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing log files: %v", errs)
	}

	return nil
}

func Global() *Logger {
	if globalLogger == nil {
		panic("logger not initialized, call New() first")
	}
	return globalLogger
}

func (l *Logger) WithScope(scope string) *Logger {
	return &Logger{
		zlog:       l.zlog.With().Str("scope", scope).Logger(),
		scope:      scope,
		appFile:    l.appFile,
		errFile:    l.errFile,
		logDir:     l.logDir,
		appLogPath: l.appLogPath,
		errLogPath: l.errLogPath,
	}
}

func (l *Logger) WithFields(fields map[string]any) *Logger {
	ctx := l.zlog.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &Logger{
		zlog:       ctx.Logger(),
		scope:      l.scope,
		appFile:    l.appFile,
		errFile:    l.errFile,
		logDir:     l.logDir,
		appLogPath: l.appLogPath,
		errLogPath: l.errLogPath,
	}
}

func (l *Logger) WithField(key string, value any) *Logger {
	return &Logger{
		zlog:       l.zlog.With().Interface(key, value).Logger(),
		scope:      l.scope,
		appFile:    l.appFile,
		errFile:    l.errFile,
		logDir:     l.logDir,
		appLogPath: l.appLogPath,
		errLogPath: l.errLogPath,
	}
}

func (l *Logger) Debug(msg string) {
	l.zlog.Debug().Str("scope", l.scope).Msg(msg)
}

func (l *Logger) Debugf(format string, args ...any) {
	l.zlog.Debug().Str("scope", l.scope).Msgf(format, args...)
}

func (l *Logger) Info(msg string) {
	l.zlog.Info().Str("scope", l.scope).Msg(msg)
}

func (l *Logger) Infof(format string, args ...any) {
	l.zlog.Info().Str("scope", l.scope).Msgf(format, args...)
}

func (l *Logger) Warn(msg string) {
	l.zlog.Warn().Str("scope", l.scope).Msg(msg)
}

func (l *Logger) Warnf(format string, args ...any) {
	l.zlog.Warn().Str("scope", l.scope).Msgf(format, args...)
}

func (l *Logger) Error(msg string) {
	l.zlog.Error().Str("scope", l.scope).Msg(msg)
}

func (l *Logger) Errorf(format string, args ...any) {
	l.zlog.Error().Str("scope", l.scope).Msgf(format, args...)
}

func (l *Logger) ErrorWithErr(err error, msg string) {
	l.zlog.Error().Err(err).Str("scope", l.scope).Msg(msg)
}

func (l *Logger) Fatal(msg string) {
	l.zlog.Fatal().Str("scope", l.scope).Msg(msg)
}

func (l *Logger) Fatalf(format string, args ...any) {
	l.zlog.Fatal().Str("scope", l.scope).Msgf(format, args...)
}

func (l *Logger) NewReader() *LogReader {
	return &LogReader{
		filePath: l.appLogPath,
	}
}

func (l *Logger) NewErrorReader() *LogReader {
	return &LogReader{
		filePath: l.errLogPath,
	}
}

func (lr *LogReader) ReadAll() ([]LogEntry, error) {
	file, err := os.Open(lr.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	var entries []LogEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var entry LogEntry
		line := scanner.Bytes()

		if err = json.Unmarshal(line, &entry); err != nil {
			continue
		}

		var raw map[string]any
		if err = json.Unmarshal(line, &raw); err == nil {
			entry.Fields = make(map[string]any)
			for k, v := range raw {
				if k != "level" && k != "time" && k != "scope" && k != "message" {
					entry.Fields[k] = v
				}
			}
		}

		entries = append(entries, entry)
	}

	if err = scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading log file: %w", err)
	}

	return entries, nil
}

func (lr *LogReader) ReadLast(n int) ([]LogEntry, error) {
	entries, err := lr.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(entries) <= n {
		return entries, nil
	}

	return entries[len(entries)-n:], nil
}

func (lr *LogReader) ReadRange(start, end int) ([]LogEntry, error) {
	entries, err := lr.ReadAll()
	if err != nil {
		return nil, err
	}

	if start < 0 {
		start = 0
	}
	if end > len(entries) {
		end = len(entries)
	}
	if start >= end {
		return []LogEntry{}, nil
	}

	return entries[start:end], nil
}

func (lr *LogReader) ReadByLevel(level string) ([]LogEntry, error) {
	entries, err := lr.ReadAll()
	if err != nil {
		return nil, err
	}

	var filtered []LogEntry
	for _, entry := range entries {
		if entry.Level == level {
			filtered = append(filtered, entry)
		}
	}

	return filtered, nil
}

func (lr *LogReader) ReadByScope(scope string) ([]LogEntry, error) {
	entries, err := lr.ReadAll()
	if err != nil {
		return nil, err
	}

	var filtered []LogEntry
	for _, entry := range entries {
		if entry.Scope == scope {
			filtered = append(filtered, entry)
		}
	}

	return filtered, nil
}

func (lr *LogReader) ReadByTimeRange(from, to time.Time) ([]LogEntry, error) {
	entries, err := lr.ReadAll()
	if err != nil {
		return nil, err
	}

	var filtered []LogEntry
	for _, entry := range entries {
		t, err := time.Parse(time.RFC3339, entry.Timestamp)
		if err != nil {
			continue
		}

		if (t.Equal(from) || t.After(from)) && (t.Equal(to) || t.Before(to)) {
			filtered = append(filtered, entry)
		}
	}

	return filtered, nil
}

// Count - возвращает количество записей в файле
func (lr *LogReader) Count() (int, error) {
	file, err := os.Open(lr.filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open log file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		count++
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error counting log entries: %w", err)
	}

	return count, nil
}

type levelWriter struct {
	writer   io.Writer
	minLevel zerolog.Level
}

func (lw *levelWriter) Write(p []byte) (n int, err error) {
	// Парсим JSON чтобы проверить уровень
	var logData map[string]interface{}
	if err := json.Unmarshal(p, &logData); err != nil {
		return len(p), nil // пропускаем невалидный JSON
	}

	level, ok := logData["level"].(string)
	if !ok {
		return len(p), nil
	}

	logLevel := parseLevel(level)
	if logLevel < lw.minLevel {
		return len(p), nil
	}

	return lw.writer.Write(p)
}

func Debug(msg string) {
	Global().Debug(msg)
}

func Debugf(format string, args ...interface{}) {
	Global().Debugf(format, args...)
}

func Info(msg string) {
	Global().Info(msg)
}

func Infof(format string, args ...interface{}) {
	Global().Infof(format, args...)
}

func Warn(msg string) {
	Global().Warn(msg)
}

func Warnf(format string, args ...interface{}) {
	Global().Warnf(format, args...)
}

func Error(msg string) {
	Global().Error(msg)
}

func Errorf(format string, args ...interface{}) {
	Global().Errorf(format, args...)
}

func ErrorWithErr(err error, msg string) {
	Global().ErrorWithErr(err, msg)
}

func Fatal(msg string) {
	Global().Fatal(msg)
}

func Fatalf(format string, args ...interface{}) {
	Global().Fatalf(format, args...)
}

func WithScope(scope string) *Logger {
	return Global().WithScope(scope)
}

func WithFields(fields map[string]interface{}) *Logger {
	return Global().WithFields(fields)
}

func WithField(key string, value interface{}) *Logger {
	return Global().WithField(key, value)
}

func Close() error {
	return Global().Close()
}

func NewReader() *LogReader {
	return Global().NewReader()
}

func NewErrorReader() *LogReader {
	return Global().NewErrorReader()
}
