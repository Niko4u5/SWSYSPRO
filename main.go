package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type Movie struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	//connect to database
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	//create the table if it doesn't exist
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS movies (id SERIAL PRIMARY KEY, name TEXT)")

	if err != nil {
		log.Fatal(err)
	}

	//create router
	router := mux.NewRouter()
	router.HandleFunc("/movies", getUsers(db)).Methods("GET")
	router.HandleFunc("/movies/id/{id}", getUser(db)).Methods("GET")
	router.HandleFunc("/movies/name/{name}", getUserByName(db)).Methods("GET")
	router.HandleFunc("/movies", createUser(db)).Methods("POST")
	router.HandleFunc("/movies/id/{id}", updateUser(db)).Methods("PUT")
	router.HandleFunc("/movies/id/{id}", deleteUser(db)).Methods("DELETE")

	//start server
	log.Fatal(http.ListenAndServe(":8080", jsonContentTypeMiddleware(router)))
}

func jsonContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// get all users
func getUsers(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT * FROM movies")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		movies := []Movie{}
		for rows.Next() {
			var movie Movie
			if err := rows.Scan(&movie.ID, &movie.Name); err != nil {
				log.Fatal(err)
			}
			movies = append(movies, movie)
		}
		if err := rows.Err(); err != nil {
			log.Fatal(err)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(movies)
	}
}

// get user by id
func getUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		var movie Movie
		err := db.QueryRow("SELECT * FROM movies WHERE id = $1", id).Scan(&movie.ID, &movie.Name)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(movie)
	}
}

// get user by name
func getUserByName(db *sql.DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		name := vars["name"]

		rows, err := db.Query("SELECT * FROM movies WHERE name ILIKE $1", name)

		movies := []Movie{}
		for rows.Next() {
			var movie Movie
			rows.Scan(&movie.ID, &movie.Name)
			if err != nil {
				log.Fatal(err)
			}
			movies = append(movies, movie)
		}
		if err := rows.Err(); err != nil {
			log.Fatal(err)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(movies)
	}
}

// create user
func createUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var movie Movie
		json.NewDecoder(r.Body).Decode(&movie)

		err := db.QueryRow("INSERT INTO movies (name) VALUES ($1) RETURNING id", movie.Name).Scan(&movie.ID)
		if err != nil {
			log.Fatal(err)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(movie)
	}
}

// update user
func updateUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var movie Movie
		json.NewDecoder(r.Body).Decode(&movie)

		vars := mux.Vars(r)
		id := vars["id"]

		_, err := db.Exec("UPDATE movies SET name = $1 WHERE id = $2", movie.Name, id)
		if err != nil {
			log.Fatal(err)
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(movie)
	}
}

// delete user
func deleteUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		var movie Movie
		err := db.QueryRow("SELECT * FROM movies WHERE id = $1", id).Scan(&movie.ID, &movie.Name)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		} else {
			_, err := db.Exec("DELETE FROM movies WHERE id = $1", id)
			if err != nil {
				//todo : fix error handling
				w.WriteHeader(http.StatusNotFound)
				return
			}

			w.WriteHeader(http.StatusNoContent)
			json.NewEncoder(w).Encode("User deleted")
		}
	}
}
