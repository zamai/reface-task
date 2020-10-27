// Integration level test, running postgres in docker container and sending http requests to router
package main_test

import (
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
	"github.com/zamai/reface-task/api"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// config vaules used in test
const (
	dbUser     = "root"
	dbPassword = "secret"
	dbName     = "cells"
	dbPort     = "5432"
)

// var db *sqlx.DB
var pool *dockertest.Pool

func TestMain(m *testing.M) {
	var err error
	pool, err = dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	code := m.Run()


	os.Exit(code)
}

func TestAddCells(t *testing.T) {
	// start app with testing DB
	db, cleanup := newTestDB("TestAddCells")
	defer cleanup()
	a := api.New(db)

	t.Run("add balls to empty cell", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/cell/add", strings.NewReader(`{"id":1,"balls":100}`))

		a.Handler.ServeHTTP(w, r)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, `{"id":1,"balls":100}`, w.Body.String())

	})
	t.Run("add balls to empty cell, more then limit", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/cell/add", strings.NewReader(`{"id":2,"balls":100000}`))

		a.Handler.ServeHTTP(w, r)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, `{"id":2,"balls":0}`, w.Body.String())

	})
	t.Run("add balls to cell with balls", func(t *testing.T) {
		// testCell = 3
		addBallsFixture(db, 3, 100)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/cell/add", strings.NewReader(`{"id":3,"balls":200}`))
		a.Handler.ServeHTTP(w, r)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, `{"id":3,"balls":300}`, w.Body.String())
	})
	t.Run("add balls to cell with balls, more then limit", func(t *testing.T) {
		addBallsFixture(db, 4, 100)
		r := httptest.NewRequest(http.MethodPost, "/cell/add", strings.NewReader(`{"id":4,"balls":100000}`))
		w := httptest.NewRecorder()

		a.Handler.ServeHTTP(w, r)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, `{"id":4,"balls":0}`, w.Body.String())
	})
	t.Run("invalid intput", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/cell/add", strings.NewReader(`{"id":5,"balls": -100}`))

		a.Handler.ServeHTTP(w, r)

		require.Equal(t, http.StatusBadRequest, w.Code)
		require.Contains(t,  w.Body.String(), "number of balls must be positive")
	})
}

func TestGetMaxCell(t *testing.T) {
	db, cleanup := newTestDB("TestAddCells")
	defer cleanup()
	a := api.New(db)

	// TODO: fix: tests are order dependent
	t.Run("no entries", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/cell/max", nil)

		a.Handler.ServeHTTP(w, r)

		require.Equal(t, http.StatusInternalServerError, w.Code)
		require.Contains(t, w.Body.String(), "no cells in the db")
	})

	t.Run("1 entry ", func(t *testing.T) {
		addBallsFixture(db, 10, 100)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/cell/max", nil)

		a.Handler.ServeHTTP(w, r)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, `{"id":10}`, w.Body.String())
	})
	t.Run("several entries with max value", func(t *testing.T) {
		addBallsFixture(db, 12, 100)
		addBallsFixture(db, 13, 10)
		addBallsFixture(db, 14, 1)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/cell/max", nil)

		a.Handler.ServeHTTP(w, r)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, `{"id":12}`, w.Body.String())
	})

}

func addBallsFixture(db *sqlx.DB, id, balls int) {
	db.MustExec("INSERT INTO cells (id, balls) VALUES ($1,$2)", id, balls)
}
// newTestDB test helper starts docker pg container, creates db with the input name, applies migrations.
// cleanup function must be called when finished working with container
func newTestDB (name string) (db *sqlx.DB, clenup func()){
	opts := dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "13-alpine",
		Env: []string{
			"POSTGRES_USER=" + dbUser,
			"POSTGRES_PASSWORD=" + dbPassword,
			"POSTGRES_DB=" + name,
		},
		ExposedPorts: []string{dbPort},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"5432": {
				{HostIP: "0.0.0.0", HostPort: dbPort},
			},
		},
	}
	// pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(&opts)
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
	dataSourceName := fmt.Sprintf(
		"host=localhost port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbPort, dbUser, dbPassword, name)

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	err = pool.Retry(func() error {
		var err error
		db, err = sqlx.Open("postgres", dataSourceName)
		if err != nil {
			return err
		}
		return db.Ping()
	})
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	mgrt, err := migrate.NewWithDatabaseInstance(
		"file://db/migration", "postgres", driver)
	if err != nil {
		log.Fatalf("Could not start NewWithDatabaseInstance: %s", err)
	}
	// for real project I would have a docker image with all of the latest migrations applied, used that image here
	err = mgrt.Up()
	if err != nil {
		log.Fatalf("Could not apply Up migrations: %s", err)
	}
	return db, func() {
		if err := pool.Purge(resource); err != nil {
			log.Fatalf("Could not purge resource: %s", err)
		}
	}
}