package gomigrator

import (
	"fmt"
	"os"
	"time"
)

type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (l *Logger) Error(msg string) {
	timestamp := time.Now().Format(time.RFC3339)
	fmt.Fprintf(os.Stderr, "[ERROR] %s %s\n", timestamp, msg)
}

func (l *Logger) Info(msg string) {
	timestamp := time.Now().Format(time.RFC3339)
	fmt.Fprintf(os.Stdout, "[INFO] %s %s\n", timestamp, msg)
}
