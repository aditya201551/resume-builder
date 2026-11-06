// Command features manages release flags using trusted database credentials.
// It deliberately exposes no browser-accessible write endpoint.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"resume-builder/backend/internal/db"
	"resume-builder/backend/internal/features"
	"resume-builder/backend/internal/features/postgres"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		// Database errors may include connection details. Keep the public output
		// limited to our own validation messages and operation names.
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer) error {
	const usage = "usage: features list | features set <key> <true|false> --actor <name>"
	var key features.Key
	var enabled bool
	var actor string
	if len(args) == 1 && args[0] == "list" {
		// Valid read command.
	} else if len(args) >= 3 && args[0] == "set" {
		key = features.Key(args[1])
		if !features.Known(key) {
			return fmt.Errorf("unknown feature: %s", key)
		}
		if args[2] != "true" && args[2] != "false" {
			return fmt.Errorf("value must be true or false")
		}
		enabled = args[2] == "true"
		options := flag.NewFlagSet("features set", flag.ContinueOnError)
		options.SetOutput(io.Discard)
		options.StringVar(&actor, "actor", "", "name recorded in the audit log")
		if err := options.Parse(args[3:]); err != nil || options.NArg() != 0 || actor == "" {
			return fmt.Errorf("%s", usage)
		}
		actor = strings.TrimSpace(actor)
		if actor == "" || len(actor) > 200 {
			return fmt.Errorf("actor must contain 1–200 bytes")
		}
	} else {
		return fmt.Errorf("%s", usage)
	}

	_ = godotenv.Load()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		return fmt.Errorf("could not connect to the feature database")
	}
	defer pool.Close()
	store := postgres.NewStore(pool)
	if args[0] == "set" {
		if err := features.NewService(store).Set(ctx, key, enabled, actor); err != nil {
			return fmt.Errorf("could not update the feature; check the key, actor, and database migrations")
		}
		_, err := fmt.Fprintf(output, "%s=%t\n", key, enabled)
		return err
	}
	// Administrative reads report database errors instead of hiding them behind
	// the fail-closed fallback intended for end-user requests.
	values, err := store.List(ctx)
	if err != nil {
		return fmt.Errorf("could not read features; check database migrations")
	}
	registered := make(map[string]bool)
	for _, definition := range features.Catalog() {
		registered[string(definition.Key)] = values[string(definition.Key)]
	}
	return json.NewEncoder(output).Encode(registered)
}
