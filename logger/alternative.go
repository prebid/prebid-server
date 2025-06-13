package logger

import (
	"fmt"
	"log/slog"
	"os"
)

type SlogWrapper struct {
	depth int
}

func (logger *SlogWrapper) Info(args ...any) {
	msg := fmt.Sprint(args...)
	slog.Info(msg)
}

func (logger *SlogWrapper) Infof(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	slog.Info(msg)
}

func (logger *SlogWrapper) Warning(args ...any) {
	msg := fmt.Sprint(args...)
	slog.Warn(msg)
}

func (logger *SlogWrapper) Warningf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	slog.Warn(msg)
}

func (logger *SlogWrapper) Warningln(args ...any) {
	msg := fmt.Sprintln(args...)
	slog.Warn(msg)
}

func (logger *SlogWrapper) Error(args ...any) {
	msg := fmt.Sprint(args...)
	slog.Error(msg)
}

func (logger *SlogWrapper) Errorf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	slog.Error(msg)
}

func (logger *SlogWrapper) Exitf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	slog.Info(msg)
	os.Exit(1)
}

func (logger *SlogWrapper) Fatal(args ...any) {
	msg := fmt.Sprint(args...)
	slog.Error(msg)
	os.Exit(1)
}

func (logger *SlogWrapper) Fatalf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	slog.Error(msg)
	os.Exit(1)
}

func ProvideAlternativeLogger(depth int) Logger {
	return &SlogWrapper{
		depth: depth,
	}
}
