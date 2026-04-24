// Package logger provides a leveled structured logger writing to stderr.
package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelOff
)

func ParseLevel(s string) Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	case "off", "silent", "none":
		return LevelOff
	default:
		return LevelInfo
	}
}

type Logger struct {
	mu    sync.Mutex
	level Level
	out   io.Writer
}

var defaultLogger = &Logger{level: LevelInfo, out: os.Stderr}

func Default() *Logger { return defaultLogger }

func SetLevel(l Level) { defaultLogger.level = l }

func SetOutput(w io.Writer) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.out = w
}

func (l *Logger) log(level Level, tag, format string, args ...any) {
	if level < l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	ts := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.out, "%s %s %s\n", ts, tag, msg)
}

func Debugf(format string, args ...any) { defaultLogger.log(LevelDebug, "DEBUG", format, args...) }
func Infof(format string, args ...any)  { defaultLogger.log(LevelInfo, "INFO ", format, args...) }
func Warnf(format string, args ...any)  { defaultLogger.log(LevelWarn, "WARN ", format, args...) }
func Errorf(format string, args ...any) { defaultLogger.log(LevelError, "ERROR", format, args...) }
