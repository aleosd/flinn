// Package main provides a command-line interface for the flinn configuration loader.
//
// Used mainly for quick local end-to-end testing, not intended to be used in real cases.
package main

import (
	"flag"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/aleosd/flinn"
	flinntoml "github.com/aleosd/flinn/source/toml"
)

type Config struct {
	Database struct {
		Host     string
		Port     int
		Username string
		Password string
	}
	API struct {
		Host string
		Port int
	}
}

var supportedLogLevels = []string{"debug", "info", "warn", "error"}

func main() {
	filePath := flag.String("config", "", "Path to configuration file")
	logLevel := flag.String("log", "debug", "Log level (debug|info|warn|error)")
	flag.Parse()
	ll := slog.LevelDebug
	level := strings.ToLower(strings.TrimSpace(*logLevel))
	if slices.Contains(supportedLogLevels, level) {
		switch level {
		case "debug":
			ll = slog.LevelDebug
		case "info":
			ll = slog.LevelInfo
		case "warn":
			ll = slog.LevelWarn
		case "error":
			ll = slog.LevelError
		}
	} else {
		log.Printf("Unknown log level '%s', using '%s' as default", *logLevel, ll.String())
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: ll})
	logger := slog.New(handler)

	if *filePath == "" {
		logger.Error("No config file provided. Use -config flag.")
		os.Exit(1)
	}
	logger.Info("loading configuration", "path", *filePath)

	fileFormat := strings.ToLower(filepath.Ext(*filePath))
	var err error
	var source flinn.Source
	switch fileFormat {
	case ".json":
		source, err = flinn.NewJSONSource(*filePath)
	case ".toml":
		source, err = flinntoml.NewTOMLSource(*filePath)
	default:
		logger.Error("Unknown file format", "format", fileFormat)
		os.Exit(1)
	}

	if err != nil {
		logger.Error("Failed to parse source", "error", err)
		os.Exit(1)
	}

	loader := flinn.NewLoader(flinn.WithSource(source), flinn.WithLogger(logger))

	var cfg Config

	fields := flinn.DefineSchema(
		flinn.FieldsGroup("database",
			flinn.String("host", &cfg.Database.Host).Default("localhost"),
			flinn.Int("port", &cfg.Database.Port).Min(1).Max(65535),
			flinn.String("username", &cfg.Database.Username).Required(),
			flinn.String("password", &cfg.Database.Password).Required(),
		),
		flinn.FieldsGroup("api",
			flinn.String("host", &cfg.API.Host).Default("localhost"),
			flinn.Int("port", &cfg.API.Port).Min(1).Max(65535),
		),
	)
	if err := loader.Load(fields); err != nil {
		if fieldErrs, ok := err.(flinn.FieldErrors); ok {
			for _, fe := range fieldErrs {
				logger.Error("config field error", "path", fe.Path, "rule", fe.Rule, "msg", fe.Msg)
			}
		} else {
			logger.Error("error loading config", "error", err.Error())
		}
		os.Exit(1)
	}

	logger.Info("Loaded config", "cfg", cfg)
}
