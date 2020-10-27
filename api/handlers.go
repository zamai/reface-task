package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
)

func (a *api) addHandler(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var add cell
	err := dec.Decode(&add)
	if err != nil {
		// parsing json handling copied from: https://www.alexedwards.net/blog/how-to-properly-parse-a-json-request-body
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		switch {
		// Catch any syntax errors in the JSON and send an error message
		// which interpolates the location of the problem to make it
		// easier for the client to fix.
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			http.Error(w, msg, http.StatusBadRequest)

		// An io.EOF error is returned by Decode() if the request body is
		// empty.
		case errors.Is(err, io.EOF):
			msg := "Request body must not be empty"
			http.Error(w, msg, http.StatusBadRequest)

		// Catch any type errors, like trying to assign a string in the
		// JSON request body to a int field in our Person struct. We can
		// interpolate the relevant field name and position into the error
		// message to make it easier for the client to fix.
		case errors.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			http.Error(w, msg, http.StatusBadRequest)

		// Otherwise default to logging the error and sending a 500 Internal
		// api Error response.
		default:
			log.Println(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}
	if add.Balls < 0 {
		msg := "number of balls must be positive"
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), dbQueryTimeOut)
	defer cancel()

	var updatedBallsCount int
	err = a.db.QueryRowxContext(ctx,
		// DB query that does the hevy lifting of updating row with number of balls and resolving conflicts of conccurent writes
		"INSERT INTO cells (id, balls) VALUES ($1, (CASE WHEN ($2::INT > $3::INT) THEN 0 ELSE $2 END))"+
			"ON CONFLICT (id) DO UPDATE SET balls = "+
			"(CASE WHEN (cells.balls+$2 > $3) THEN 0 ELSE cells.balls+$2 END)"+ // check the max balls size on insert
			"WHERE cells.id = $1 "+
			"RETURNING cells.balls", add.ID, add.Balls, maxBalls).Scan(&updatedBallsCount)
	if err != nil {
		log.Printf("error updating db: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resp := cell{
		ID:    add.ID,
		Balls: updatedBallsCount,
	}

	b, err := json.Marshal(resp)
	if err != nil {
		log.Printf("could not marshal into json: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	_, err = w.Write(b)
	if err != nil {
		log.Printf("could not write response bytes: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

type maxCellResp struct {
	ID int `json:"id" db:"id"` // id of the cell
}

func (a *api) maxHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), dbQueryTimeOut)
	defer cancel()

	var resp maxCellResp
	err := a.db.QueryRowxContext(ctx, `SELECT id FROM cells ORDER BY balls DESC LIMIT 1;`).StructScan(&resp)
	if err != nil {
		if err == sql.ErrNoRows {
			// TODO: clarify which responsse should user get in this case, maybe {"id":0}
			msg := "no cells in the db"
			http.Error(w, msg, http.StatusInternalServerError)
		}
		log.Printf("error finding cell with max balls size: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(resp)
	if err != nil {
		log.Printf("could not marshal into json: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	_, err = w.Write(b)
	if err != nil {
		log.Printf("could not write response bytes: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
