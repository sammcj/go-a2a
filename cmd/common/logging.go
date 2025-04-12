package common

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// LogLevel represents the level of logging.
type LogLevel int

const (
	// LogLevelDebug represents debug level logging.
	LogLevelDebug LogLevel = iota
	// LogLevelInfo represents info level logging.
	LogLevelInfo
	// LogLevelWarn represents warning level logging.
	LogLevelWarn
	// LogLevelError represents error level logging.
	LogLevelError
	// LogLevelFatal represents fatal level logging.
	LogLevelFatal
)

// Logger represents a logger.
type Logger struct {
	debugLogger *log.Logger
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	fatalLogger *log.Logger
	level       LogLevel
}

// NewLogger creates a new logger.
func NewLogger(out io.Writer, level string) *Logger {
	// Parse log level
	logLevel := parseLogLevel(level)

	// Create loggers
	debugLogger := log.New(out, "DEBUG: ", log.Ldate|log.Ltime)
	infoLogger := log.New(out, "INFO: ", log.Ldate|log.Ltime)
	warnLogger := log.New(out, "WARN: ", log.Ldate|log.Ltime)
	errorLogger := log.New(out, "ERROR: ", log.Ldate|log.Ltime)
	fatalLogger := log.New(out, "FATAL: ", log.Ldate|log.Ltime)

	return &Logger{
		debugLogger: debugLogger,
		infoLogger:  infoLogger,
		warnLogger:  warnLogger,
		errorLogger: errorLogger,
		fatalLogger: fatalLogger,
		level:       logLevel,
	}
}

// Debug logs a debug message.
func (l *Logger) Debug(format string, v ...interface{}) {
	if l.level <= LogLevelDebug {
		l.debugLogger.Printf(format, v...)
	}
}

// Info logs an info message.
func (l *Logger) Info(format string, v ...interface{}) {
	if l.level <= LogLevelInfo {
		l.infoLogger.Printf(format, v...)
	}
}

// Warn logs a warning message.
func (l *Logger) Warn(format string, v ...interface{}) {
	if l.level <= LogLevelWarn {
		l.warnLogger.Printf(format, v...)
	}
}

// Error logs an error message.
func (l *Logger) Error(format string, v ...interface{}) {
	if l.level <= LogLevelError {
		l.errorLogger.Printf(format, v...)
	}
}

// Fatal logs a fatal message and exits.
func (l *Logger) Fatal(format string, v ...interface{}) {
	if l.level <= LogLevelFatal {
		l.fatalLogger.Printf(format, v...)
		os.Exit(1)
	}
}

// parseLogLevel parses a log level string.
func parseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return LogLevelDebug
	case "info":
		return LogLevelInfo
	case "warn", "warning":
		return LogLevelWarn
	case "error":
		return LogLevelError
	case "fatal":
		return LogLevelFatal
	default:
		fmt.Printf("Unknown log level: %s, defaulting to info\n", level)
		return LogLevelInfo
	}
}

// DefaultLogger returns a default logger.
func DefaultLogger() *Logger {
	return NewLogger(os.Stdout, "info")
}
