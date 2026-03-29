package main

import (
	"fmt"

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
	fmt.Println("Flinning!!!")
	loader := flinn.NewLoader()

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
	loader.Load(fields)
}
