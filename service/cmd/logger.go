package cmd

import (
	"io"
	"os"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

func initLogger(config *LogConfig) error {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.TimeFieldFormat = time.RFC3339

	level, err := parseLevel(config.Level)
	if err != nil {
		return err
	}

	var w io.Writer
	if config.Output == "stdout" || config.Output == "" {
		w = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.DateTime}
	} else {
		file, err := os.OpenFile(config.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
		if err != nil {
			return err
		}

		if !config.DisableStdout && isatty.IsTerminal(os.Stdout.Fd()) {
			w = zerolog.MultiLevelWriter(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.DateTime}, file)
		} else {
			w = file
		}
		// TODO: close file
	}
	logger := zerolog.New(w).Level(level).With().Timestamp().Caller().Logger()
	log = &logger
	return nil
}

func parseLevel(level string) (zerolog.Level, error) {
	if level == "" {
		return zerolog.InfoLevel, nil
	}
	return zerolog.ParseLevel(level)
}
