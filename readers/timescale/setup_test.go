// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package timescale_test contains tests for PostgreSQL repository
// implementations.
package timescale_test

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	tsWriter "github.com/absmach/magistrala/consumers/writers/timescale"
	pgclient "github.com/absmach/magistrala/pkg/postgres"
	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	"github.com/jmoiron/sqlx"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var db *sqlx.DB

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	container, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "timescale/timescaledb",
		Tag:        "2.19.3-pg16-oss",
		Env: []string{
			"POSTGRES_USER=test",
			"POSTGRES_PASSWORD=test",
			"POSTGRES_DB=test",
			"listen_addresses = '*'",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start container: %s", err)
	}

	port := container.GetPort("5432/tcp")
	url := fmt.Sprintf("host=localhost port=%s user=test dbname=test password=test sslmode=disable", port)

	if err = pool.Retry(func() error {
		db, err = sqlx.Open("pgx", url)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	dbConfig := pgclient.Config{
		Host:        "localhost",
		Port:        port,
		User:        "test",
		Pass:        "test",
		Name:        "test",
		SSLMode:     "disable",
		SSLCert:     "",
		SSLKey:      "",
		SSLRootCert: "",
		Pool: pgclient.PoolConfig{
			MaxConnLifetime:       1 * time.Hour,
			MaxConnLifetimeJitter: time.Duration(0),
			MaxConnIdleTime:       15 * time.Minute,
			MaxConns:              5,
			MinConns:              1,
			MinIdleConns:          1,
			HealthCheckPeriod:     1 * time.Minute,
		},
	}

	if db, err = pgclient.Setup(dbConfig, *tsWriter.Migration()); err != nil {
		log.Fatalf("Could not setup test DB connection: %s", err)
	}

	code := m.Run()

	// Defers will not be run when using os.Exit
	db.Close()
	if err = pool.Purge(container); err != nil {
		log.Fatalf("Could not purge container: %s", err)
	}

	os.Exit(code)
}
