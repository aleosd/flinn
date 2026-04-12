package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/aleosd/flinn"
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

func main() {
	fmt.Println("Loading configuration...")

	filPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()
	source, err := flinn.NewJSONSource(*filPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing source: %v\n", err)
		os.Exit(1)
	}
	loader := flinn.NewLoader(flinn.WithSource(source))

	var cfg Config

	fields := []flinn.Field{
		flinn.Group("database", []flinn.Field{
			flinn.String("host", &cfg.Database.Host),
			flinn.Int("port", &cfg.Database.Port),
			flinn.String("username", &cfg.Database.Username),
			flinn.String("password", &cfg.Database.Password),
		}),
		flinn.Group("api", []flinn.Field{
			flinn.String("host", &cfg.API.Host),
			flinn.Int("port", &cfg.API.Port),
		}),
	}
	if err := loader.Load(fields); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(cfg)
	fmt.Println("Done!")
}
