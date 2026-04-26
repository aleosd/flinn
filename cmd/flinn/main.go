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
	filPath := flag.String("config", "", "Path to configuration file")
	logLevel := flag.String("log", "debug", "Log level (debug|info|warn|error)")
	flag.Parse()
	ll := slog.LevelInfo
	if logLevel != nil {
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
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: ll})
	logger := slog.New(handler)

	logger.Info("loading configuration", "path", *filPath)

	fileFormat := strings.ToLower(filepath.Ext(*filPath))
	var err error
	var source flinn.Source
	switch fileFormat {
	case ".json":
		source, err = flinn.NewJSONSource(*filPath)
	case ".toml":
		source, err = flinntoml.NewTOMLSource(*filPath)
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
			flinn.String("host", &cfg.Database.Host),
			flinn.Int("port", &cfg.Database.Port),
			flinn.String("username", &cfg.Database.Username),
			flinn.String("password", &cfg.Database.Password),
		),
		flinn.FieldsGroup("api",
			flinn.String("host", &cfg.API.Host),
			flinn.Int("port", &cfg.API.Port),
		),
	)
	if err := loader.Load(fields); err != nil {
		logger.Error("error loading config", "error", err.Error())
		os.Exit(1)
	}

	logger.Info("Loaded config", "cfg", cfg)
}
