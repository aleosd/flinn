package flinn

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestConfig struct {
	Database struct {
		Host string
		Port int
	}
	RootURL string
}

func TestLoader_LoadsFromEnv(t *testing.T) {
	t.Run("without source", func(t *testing.T) {
		// arrange
		port := 8080
		host := "my.db.host"
		rootURL := "https://example.com"
		t.Setenv("PORT", strconv.Itoa(port))
		t.Setenv("HOST", host)
		t.Setenv("ROOT_URL", rootURL)
		var cfg TestConfig
		fields := []Field{
			Group("database", []Field{
				String("host", &cfg.Database.Host, Env("HOST")),
				Int("port", &cfg.Database.Port, Env("PORT")),
			}),
			String("root_url", &cfg.RootURL, Env("ROOT_URL")),
		}
		loader := NewLoader()

		//act
		err := loader.Load(fields)

		//assert
		require.NoError(t, err)
		assert.Equal(t, host, cfg.Database.Host)
		assert.Equal(t, port, cfg.Database.Port)
		assert.Equal(t, rootURL, cfg.RootURL)
	})

	t.Run("with prefix", func(t *testing.T) {
		// arrange
		port := 8080
		host := "my.db.host"
		rootURL := "https://example.com"
		t.Setenv("FL_DB_PORT", strconv.Itoa(port))
		t.Setenv("FL_DB_HOST", host)
		t.Setenv("FL_ROOT_URL", rootURL)
		var cfg TestConfig
		fields := []Field{
			Group("database", []Field{
				String("host", &cfg.Database.Host, Env("HOST")),
				Int("port", &cfg.Database.Port, Env("PORT")),
			}, Env("DB")),
			String("root_url", &cfg.RootURL, Env("ROOT_URL")),
		}
		loader := NewLoader(WithEnvPrefix("FL"))

		//act
		err := loader.Load(fields)

		//assert
		require.NoError(t, err)
		assert.Equal(t, host, cfg.Database.Host)
		assert.Equal(t, port, cfg.Database.Port)
		assert.Equal(t, rootURL, cfg.RootURL)
	})
}
