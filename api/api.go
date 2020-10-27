package api

import (
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"net/http"
	"time"
)

const (
	dbQueryTimeOut = 1 * time.Second
	maxBalls       = 10000
)

// api serves HTTP requests for cells application.
type api struct {
	Handler http.Handler
	db      *sqlx.DB
}

// New creates a new HTTP server and set up routing.
func New(db *sqlx.DB) *api {
	a := &api{db: db}
	r := mux.NewRouter()
	r.HandleFunc("/cell/add", a.addHandler).Methods("POST")
	r.HandleFunc("/cell/max", a.maxHandler).Methods("GET")

	a.Handler = r

	return a
}

// cell struct representing request of adding balls to the cell
type cell struct {
	ID    int `json:"id" db:"id"`       // id of the cell
	Balls int `json:"balls" db:"balls"` // number of balls to add into the cell
}
