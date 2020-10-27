package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/zamai/reface-task/api"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

// in production make these be variables from the environment or config
const (
	host       = "localhost"
	port       = 5432
	user       = "root"
	password   = "secret"
	dbname     = "cells"
	serverPort = "8080"
)

// api biolerplate is coppied from gorrila mux example
func main() {
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15,
		"the duration for which the App gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	dataSourceName := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	// sqlx.Open()
	db, err := sqlx.Open("postgres", dataSourceName)
	if err != nil {
		log.Fatalf("count not connect to database: %+v", err)
	}

	a := api.New(db)
	srv := &http.Server{
		Addr:         "0.0.0.0:" + serverPort,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      a.Handler,
	}
	// Run our App in a goroutine so that it doesn't block.
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()
	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)

	log.Println("shutting down")
	os.Exit(0)
}
