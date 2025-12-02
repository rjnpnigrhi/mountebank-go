package util

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Logger wraps logrus with additional functionality
type Logger struct {
	*logrus.Logger
	scope string
	hook  *LogHook
}

// NewLogger creates a new logger instance
func NewLogger(level string) *Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Set log level
	switch strings.ToLower(level) {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn", "warning":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	logger.SetOutput(os.Stdout)

	hook := &LogHook{
		Entries: make([]LogEntry, 0),
	}
	logger.AddHook(hook)

	return &Logger{
		Logger: logger,
		scope:  "",
		hook:   hook,
	}
}

// WithScope creates a new logger with a scope prefix
func (l *Logger) WithScope(scope string) *Logger {
	return &Logger{
		Logger: l.Logger,
		scope:  scope,
		hook:   l.hook,
	}
}

// ChangeScope changes the scope of the current logger
func (l *Logger) ChangeScope(scope string) {
	l.scope = scope
}

// ScopePrefix returns the current scope prefix
func (l *Logger) ScopePrefix() string {
	return l.scope
}

// formatMessage adds scope prefix to message
func (l *Logger) formatMessage(msg string) string {
	if l.scope != "" {
		return fmt.Sprintf("[%s] %s", l.scope, msg)
	}
	return msg
}

// Debug logs a debug message
func (l *Logger) Debug(args ...interface{}) {
	l.Logger.Debug(l.formatMessage(fmt.Sprint(args...)))
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Logger.Debug(l.formatMessage(fmt.Sprintf(format, args...)))
}

// Info logs an info message
func (l *Logger) Info(args ...interface{}) {
	l.Logger.Info(l.formatMessage(fmt.Sprint(args...)))
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logger.Info(l.formatMessage(fmt.Sprintf(format, args...)))
}

// Warn logs a warning message
func (l *Logger) Warn(args ...interface{}) {
	l.Logger.Warn(l.formatMessage(fmt.Sprint(args...)))
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Logger.Warn(l.formatMessage(fmt.Sprintf(format, args...)))
}

// Error logs an error message
func (l *Logger) Error(args ...interface{}) {
	l.Logger.Error(l.formatMessage(fmt.Sprint(args...)))
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logger.Error(l.formatMessage(fmt.Sprintf(format, args...)))
}

// LogEntry represents a log entry
type LogEntry struct {
	Level     string `json:"level"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// LogHook captures logs in memory
type LogHook struct {
	Entries []LogEntry
}

// Levels returns the supported log levels
func (h *LogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire is called when a log entry is created
func (h *LogHook) Fire(entry *logrus.Entry) error {
	h.Entries = append(h.Entries, LogEntry{
		Level:     entry.Level.String(),
		Message:   entry.Message,
		Timestamp: entry.Time.Format(time.RFC3339),
	})
	return nil
}

// GetEntries returns the captured log entries
func (l *Logger) GetEntries(startIndex, endIndex int) []LogEntry {
	if l.hook == nil {
		return []LogEntry{}
	}

	entries := l.hook.Entries
	total := len(entries)

	if startIndex < 0 {
		startIndex = 0
	}
	if startIndex > total {
		startIndex = total
	}

	if endIndex < 0 || endIndex > total {
		endIndex = total
	}

	if startIndex > endIndex {
		return []LogEntry{}
	}

	return entries[startIndex:endIndex]
}
