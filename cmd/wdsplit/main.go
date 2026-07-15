// Command wdsplit runs the wdsplit server and its maintenance subcommands.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	spa "github.com/andrey-vk/wdsplit"
	"github.com/andrey-vk/wdsplit/internal/config"
	"github.com/andrey-vk/wdsplit/internal/crypto"
	"github.com/andrey-vk/wdsplit/internal/db"
	"github.com/andrey-vk/wdsplit/internal/settings"
	"github.com/andrey-vk/wdsplit/internal/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("wdsplit exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return errors.New("usage: wdsplit <migrate|serve|genkey>")
	}
	cmd := os.Args[1]

	if cmd == "genkey" {
		key, err := crypto.GenerateKey()
		if err != nil {
			return fmt.Errorf("generate key: %w", err)
		}
		fmt.Println(key)
		return nil
	}

	if cmd != "migrate" && cmd != "serve" {
		return fmt.Errorf("unknown command %q; usage: wdsplit <migrate|serve|genkey>", cmd)
	}

	dbPath := os.Getenv("WDSPLIT_DB")
	if dbPath == "" {
		dbPath = "/data/wdsplit.sqlite3"
	}

	conn, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() {
		if cerr := conn.Close(); cerr != nil {
			slog.Error("close database", "error", cerr)
		}
	}()

	switch cmd {
	case "migrate":
		slog.Info("migrations applied")
		return nil

	case "serve":
		host := os.Getenv("WDSPLIT_HOST")
		if host == "" {
			host = "0.0.0.0"
		}
		port := os.Getenv("WDSPLIT_PORT")
		if port == "" {
			port = "8080"
		}

		cfg, err := config.Load(context.Background(), settings.NewSQLStore(conn), nil)
		if err != nil {
			return fmt.Errorf("load settings: %w", err)
		}

		distFS, err := spa.DistFS()
		if err != nil {
			return fmt.Errorf("load embedded frontend: %w", err)
		}

		srv := &http.Server{
			Addr:         host + ":" + port,
			Handler:      web.NewServer(cfg, distFS),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		slog.Info("wdsplit listening", "addr", srv.Addr, "db", dbPath)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	}

	return nil
}
