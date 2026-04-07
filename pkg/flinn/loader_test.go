package flinn

import (
	"fmt"
	"strconv"
	"strings"
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

type inMemorySource struct {
	data map[string]any
}

func (s inMemorySource) Get(pathSegments []string) (string, bool, error) {
	var value any = s.data
	var ok bool
	for _, segment := range pathSegments {
		s, ok := value.(map[string]any)
		if !ok {
			return "", false, nil
		}
		value, ok = s[segment]
		if !ok {
			return "", false, nil
		}
	}

	str, ok := stringify(value)
	if !ok {
		return "", false, fmt.Errorf("value at %q is not a string-convertible type", strings.Join(pathSegments, "."))
	}
	return str, true, nil
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

	t.Run("without group prefix", func(t *testing.T) {
		// arrange
		port := 8080
		host := "my.db.host"
		rootURL := "https://example.com"
		t.Setenv("FL_PORT", strconv.Itoa(port))
		t.Setenv("FL_HOST", host)
		t.Setenv("FL_ROOT_URL", rootURL)
		var cfg TestConfig
		fields := []Field{
			Group("database", []Field{
				String("host", &cfg.Database.Host, Env("HOST")),
				Int("port", &cfg.Database.Port, Env("PORT")),
			}),
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

	t.Run("with auto env", func(t *testing.T) {
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
				String("host", &cfg.Database.Host),
				Int("port", &cfg.Database.Port),
			}),
			String("root_url", &cfg.RootURL),
		}
		loader := NewLoader(WithAutoEnv())

		//act
		err := loader.Load(fields)

		//assert
		require.NoError(t, err)
		assert.Equal(t, host, cfg.Database.Host)
		assert.Equal(t, port, cfg.Database.Port)
		assert.Equal(t, rootURL, cfg.RootURL)
	})
}

func TestLoader_LoadsFromSource(t *testing.T) {
	var sourceData = map[string]any{
		"database": map[string]any{
			"host": "my.db.host",
			"port": 8080,
		},
		"root_url": "https://example.com",
	}

	t.Run("loads data from source", func(t *testing.T) {
		// arrange
		var cfg TestConfig
		fields := []Field{
			Group("database", []Field{
				String("host", &cfg.Database.Host),
				Int("port", &cfg.Database.Port),
			}),
			String("root_url", &cfg.RootURL),
		}
		source := inMemorySource{data: sourceData}
		loader := NewLoader(WithSource(source))

		// act
		err := loader.Load(fields)

		// assert
		require.NoError(t, err)
		assert.Equal(t, "my.db.host", cfg.Database.Host)
		assert.Equal(t, 8080, cfg.Database.Port)
		assert.Equal(t, "https://example.com", cfg.RootURL)

	})

	t.Run("loads data from source with to snakecase conversion", func(t *testing.T) {
		// arrange
		var cfg TestConfig
		fields := []Field{
			Group("Database", []Field{
				String("Host", &cfg.Database.Host),
				Int("Port", &cfg.Database.Port),
			}),
			String("RootURL", &cfg.RootURL),
		}
		source := inMemorySource{data: sourceData}
		loader := NewLoader(WithSource(source))

		// act
		err := loader.Load(fields)

		// assert
		require.NoError(t, err)
		assert.Equal(t, "my.db.host", cfg.Database.Host)
		assert.Equal(t, 8080, cfg.Database.Port)
		assert.Equal(t, "https://example.com", cfg.RootURL)

	})

	t.Run("loads data based on file key option", func(t *testing.T) {
		// arrange
		var cfg TestConfig
		fields := []Field{
			Group("DB", []Field{
				String("host_name", &cfg.Database.Host, FileKey("host")),
				Int("Port", &cfg.Database.Port),
			}, FileKey("database")),
			String("root_endpoint", &cfg.RootURL, FileKey("root_url")),
		}
		source := inMemorySource{data: sourceData}
		loader := NewLoader(WithSource(source))

		// act
		err := loader.Load(fields)

		// assert
		require.NoError(t, err)
		assert.Equal(t, "my.db.host", cfg.Database.Host)
		assert.Equal(t, 8080, cfg.Database.Port)
		assert.Equal(t, "https://example.com", cfg.RootURL)
	})
}

func TestLoader_DefaultOption(t *testing.T) {
	t.Run("loads default value", func(t *testing.T) {
		// arrange
		var cfg TestConfig
		url := "http://example.com"
		fields := []Field{
			String("host", &cfg.RootURL, Default(url)),
		}
		loader := NewLoader()

		// act
		err := loader.Load(fields)

		// assert
		require.NoError(t, err)
		assert.Equal(t, url, cfg.RootURL)
	})

	t.Run("env var has precedence over default value", func(t *testing.T) {
		// arrange
		var cfg TestConfig
		url := "http://example.com"
		envURL := "https://my-domain.com"
		fields := []Field{
			String("root_url", &cfg.RootURL, Default(url), Env("ROOT_URL")),
		}
		loader := NewLoader()
		t.Setenv("ROOT_URL", envURL)

		// act
		err := loader.Load(fields)

		// assert
		require.NoError(t, err)
		assert.Equal(t, envURL, cfg.RootURL)
	})

	t.Run("source var has precedence over default value", func(t *testing.T) {
		// arrange
		var cfg TestConfig
		url := "http://example.com"
		sourceURL := "https://my-domain.com"
		var sourceData = map[string]any{
			"root_url": sourceURL,
		}
		fields := []Field{
			String("root_url", &cfg.RootURL, Default(url)),
		}
		loader := NewLoader(WithSource(inMemorySource{data: sourceData}))

		// act
		err := loader.Load(fields)

		// assert
		require.NoError(t, err)
		assert.Equal(t, sourceURL, cfg.RootURL)
	})
}
