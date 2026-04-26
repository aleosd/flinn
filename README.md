# Flinn

**Flinn** is a declarative, type-safe configuration loader for Go. It resolves
values from multiple sources — environment variables, JSON files, TOML files —
with support for defaults, required fields, and nested config structs. All
errors are collected before returning, so you get a complete picture of what's
misconfigured rather than failing on the first problem.

______________________________________________________________________

## Features

- **Declarative field definitions** — describe your config schema with typed
  constructors (`String`, `Int`, `FieldsGroup`)
- **Multiple sources** — environment variables, JSON, and TOML files, resolved
  in a defined precedence order
- **Fail-complete error collection** — all field errors are gathered before
  returning, reported together as `FieldErrors`
- **Nested config structs** — `FieldsGroup` composes fields into namespaced path and
  env-prefix segments
- **Auto env-var naming** — `WithAutoEnv()` derives env variable names from
  field names via `toSnakeCase` → uppercase
- **Extensible** — implement the single-method `Source` interface to add any
  config backend

______________________________________________________________________

## Installation

```bash
go get github.com/aleosd/flinn
```

For the TOML source (optional, separate module):

```bash
go get github.com/aleosd/flinn/source/toml
```

______________________________________________________________________

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/aleosd/flinn"
)

type Config struct {
    Database struct {
        Host     string
        Port     int
        Password string
    }
    API struct {
        Host string
        Port int
    }
}

func main() {
    source, err := flinn.NewJSONSource("config.json")
    if err != nil {
        log.Fatal(err)
    }

    loader := flinn.NewLoader(
        flinn.WithSource(source),
        flinn.WithEnvPrefix("APP"),
        flinn.WithAutoEnv(),
    )

    var cfg Config
    fields := flinn.DefineSchema(
        flinn.FieldsGroup("database",
            flinn.String("host", &cfg.Database.Host).Default("localhost"),
            flinn.Int("port", &cfg.Database.Port).Default(5432),
            flinn.String("password", &cfg.Database.Password).Env("DB_PASSWORD").Required(),
        ).EnvPrefix("DB"),
        flinn.FieldsGroup("api",
            flinn.String("host", &cfg.API.Host).Default("0.0.0.0"),
            flinn.Int("port", &cfg.API.Port).Default(8080),
        ),
    )

    if err := loader.Load(fields); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }

    fmt.Printf("%+v\n", cfg)
}
```

With `WithEnvPrefix("APP")`, `WithAutoEnv()`, and `EnvPrefix("DB")` on the database
group:

- `cfg.Database.Host` ← env `APP_DB_HOST` → JSON `database.host` →
  `"localhost"`
- `cfg.Database.Password` ← env `DB_PASSWORD` (explicit) → JSON
  `database.password` → error (required)
- `cfg.API.Port` ← env `APP_PORT` → JSON `api.port` → `8080`

______________________________________________________________________

## Field Types

### `String`

```go
flinn.String("fieldName", &dest).Required().Default("value")
```

### `Int`

```go
flinn.Int("fieldName", &dest).Min(1).Max(100).Default(80)
```

Returns a `NumericField`, which supports range-based validation.

### `Float`

```go
flinn.Float("fieldName", &dest).Min(0.0).Max(1.0)
```

Returns a `NumericField`, which supports range-based validation.

### `FieldsGroup`

Groups nest fields under a named path segment. They do not hold a value
themselves.

```go
flinn.FieldsGroup("database",
    flinn.String("host", &cfg.Database.Host),
    flinn.Int("port", &cfg.Database.Port),
).EnvPrefix("DB")
```

A group's path segment (`database`) is used when looking up values in a source
file. Its env-prefix contribution is controlled by the `EnvPrefix` method.

______________________________________________________________________

## Field Options (Methods)

Fields and Groups use a builder pattern for configuration:

| Method            | Description                                                                                                           |
| ----------------- | --------------------------------------------------------------------------------------------------------------------- |
| `Env("VAR_NAME")` | Explicit environment variable name for a Field.                                                                      |
| `EnvPrefix("P")`  | Sets the environment variable prefix segment contributed by a Group.                                                  |
| `FileKey("key")`  | Override the key used when reading from a source file (defaults to `toSnakeCase(name)`).                              |
| `Default(value)`  | Value to use if nothing is found in env or source. Must match the field's type.                                       |
| `Required()`      | Return an error if no value is found from any source and no default is set.                                           |
| `Min(v)`          | (Numeric only) Ensures value is >= v.                                                                                 |
| `Max(v)`          | (Numeric only) Ensures value is <= v.                                                                                 |
| `AddValidator(f)` | Adds a custom validation function `func(T) error`.                                                                    |

______________________________________________________________________

## Value Resolution Order

For each leaf field, Flinn resolves in this precedence order (highest to
lowest):

1. **Environment variable** — looked up by key from `Env()` or auto-generated
   by `WithAutoEnv()`
1. **Config source** — looked up by path in the registered `Source` (JSON,
   TOML, …)
1. **Default value** — the value passed to `Default()`
1. **Error** — if `Required()` is set and nothing resolved, a `FieldError` is
   added

______________________________________________________________________

## Loader Options

### `WithSource(source)`

Register a config file source. Flinn ships with a JSON source; TOML is
available as a separate module.

```go
source, _ := flinn.NewJSONSource("config.json")
flinn.NewLoader(flinn.WithSource(source))
```

### `WithEnvPrefix(prefix)`

Prepend a global string to all env variable names.

```go
flinn.NewLoader(flinn.WithEnvPrefix("MYAPP"))
// field "host" → env "MYAPP_HOST" (if WithAutoEnv is enabled)
```

### `WithAutoEnv()`

Enable automatic env variable derivation for leaf fields. If `Env()` is not
explicitly set, the env key is computed from the field's path segments.
**Groups contribute to the env prefix only if `EnvPrefix()` is called.**

```go
// Without EnvPrefix() on the group:
flinn.NewLoader(flinn.WithEnvPrefix("APP"), flinn.WithAutoEnv())
// FieldsGroup("database", flinn.String("host", ...)) → env APP_HOST

// With EnvPrefix() on the group:
// FieldsGroup("database", flinn.String("host", ...)).EnvPrefix("DB") → env APP_DB_HOST
```

### `WithLogger(logger)`

Attach an `*slog.Logger` for debug/warn output during loading.

```go
flinn.NewLoader(flinn.WithLogger(slog.Default()))
```

______________________________________________________________________

## Sources

### JSON (built-in)

```go
source, err := flinn.NewJSONSource("path/to/config.json")
```

The root must be a JSON object. Nested objects map to `FieldsGroup` path segments.

```json
{
  "database": {
    "host": "localhost",
    "port": 5432
  },
  "debug": true
}
```

### TOML (separate module)

```bash
go get github.com/aleosd/flinn/source/toml
```

```go
import flinntoml "github.com/aleosd/flinn/source/toml"

source, err := flinntoml.NewTOMLSource("path/to/config.toml")
loader := flinn.NewLoader(flinn.WithSource(source))
```

The root must be a TOML table. Nested tables map to `FieldsGroup` path segments. All
scalar TOML types are supported: strings, integers, floats, booleans, and
datetime types (offset datetime, local datetime, local date, local time).

______________________________________________________________________

## Error Handling

`loader.Load` returns a `flinn.FieldErrors` value (which implements `error`)
when one or more fields fail to resolve or validate. You can type-assert to
inspect individual errors:

```go
if err := loader.Load(fields); err != nil {
    if fieldErrs, ok := err.(flinn.FieldErrors); ok {
        for _, fe := range fieldErrs {
            fmt.Printf("field %q: [%s] %s\n", fe.Path, fe.Rule, fe.Msg)
        }
    }
    os.Exit(1)
}
```

Each `FieldError` carries:

| Field   | Description                                                                              |
| ------- | ---------------------------------------------------------------------------------------- |
| `Path`  | Dot-separated path to the field, e.g. `"database.port"`                                  |
| `Rule`  | The rule that failed: `"required"`, `"parse"`, `"resolve"`, or `"validate"`              |
| `Value` | The raw string value that was present (may be `nil` for required/missing errors)         |
| `Msg`   | Human-readable error message                                                             |

______________________________________________________________________

## Implementing a Custom Source

Implement the `flinn.Source` interface to support any config backend:

```go
type Source interface {
    Get(path []string) (string, bool, error)
}
```

`path` is a slice of key segments corresponding to the field's nested position
(e.g. `["database", "host"]`). Return `("", false, nil)` when the key is
absent, `(value, true, nil)` when found, or `("", false, err)` on a retrieval
error.

______________________________________________________________________

## Auto Key Naming

When no `Env()` or `FileKey()` option is given, Flinn derives keys
automatically using `toSnakeCase`:

| Field name    | Snake case key  | Auto env var (with `WithAutoEnv`) |
| ------------- | --------------- | --------------------------------- |
| `host`        | `host`          | `HOST`                            |
| `dbHost`      | `db_host`       | `DB_HOST`                         |
| `APIPort`     | `api_port`      | `API_PORT`                        |
| `MyFieldName` | `my_field_name` | `MY_FIELD_NAME`                   |

The snake case key is used for source file lookups; the uppercased form is the
env variable name.

______________________________________________________________________

## Contributing

Bug reports and pull requests are welcome. Run the full quality suite before
submitting:

```bash
make verify
```

This runs `gofmt`, `go vet`, `golangci-lint`, `go test`, and `govulncheck`
across all modules.
