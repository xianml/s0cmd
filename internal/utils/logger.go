package utils

import (
	"log"
	"os"
)

// Logger is a simple logging utility.
type Logger struct {
	*log.Logger
}

// NewLogger creates a new Logger instance.
func NewLogger() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// Info logs an informational message.
func (l *Logger) Info(msg string) {
	l.Printf("INFO: %s", msg)
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string) {
	l.Printf("DEBUG: %s", msg)
}

// Error logs an error message.
func (l *Logger) Error(msg string) {
	l.Printf("ERROR: %s", msg)
}
