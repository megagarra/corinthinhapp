package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/gorilla/handlers"
    "github.com/gorilla/mux"
    _ "github.com/lib/pq"
)

const (
    DBHost     = "DATABASE_HOST"
    DBPort     = "DATABASE_PORT"
    DBUser     = "DATABASE_USER"
    DBPassword = "DATABASE_PASSWORD"
    DBName     = "DATABASE_NAME"
)

var db *sql.DB

type Player struct {
    ID            int       `json:"id"`
    Name          string    `json:"name"`
    PresenceDate  time.Time `json:"presence_date"`
    PresenceState string    `json:"presence_state"`
}

func init() {
    dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
        os.Getenv(DBHost),
        os.Getenv(DBPort),
        os.Getenv(DBUser),
        os.Getenv(DBPassword),
        os.Getenv(DBName),
    )

    var err error
    db, err = sql.Open("postgres", dsn)
    if err != nil {
        panic(err)
    }
}

func main() {
    router := mux.NewRouter()

    router.HandleFunc("/presences", getPresences).Methods("GET")
    router.HandleFunc("/players/{id}", getPlayer).Methods("GET")
    router.HandleFunc("/players", createPlayer).Methods("POST")
    router.HandleFunc("/players/{id}", updatePlayer).Methods("PUT")
    router.HandleFunc("/players/{id}", deletePlayer).Methods("DELETE")
    router.HandleFunc("/deleteAll", deleteAll).Methods("DELETE")

    // Use o middleware handlers.CORS para adicionar os cabeçalhos CORS necessários
    headers := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"})
    methods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE"})
    origins := handlers.AllowedOrigins([]string{"http://localhost:9000"})
    log.Fatal(http.ListenAndServe(":8080", handlers.CORS(headers, methods, origins)(router)))
}

func getPresences(w http.ResponseWriter, r *http.Request) {
    rows, err := db.Query("SELECT id, name, presence_date, presence_status FROM players")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    players := make([]Player, 0)
    for rows.Next() {
        var p Player
        err := rows.Scan(&p.ID, &p.Name, &p.PresenceDate, &p.PresenceState)
        if err != nil {
            http.Error(w, err.Error(),
                http.StatusInternalServerError)
            return
        }
        players = append(players, p)
    }

    if err = rows.Err(); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(players)
}

func getPlayer(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    playerID := vars["id"]

    row := db.QueryRow("SELECT id, name, presence FROM players WHERE id = ?", playerID)

	var p Player
	err := row.Scan(&p.ID, &p.Name, &p.PresenceDate, &p.PresenceState)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}


func createPlayer(w http.ResponseWriter, r *http.Request) {
	var p Player
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	res, err := db.Exec("INSERT INTO players (name, presence_date, presence_status) VALUES (?, ?, ?)", p.Name, p.PresenceDate, p.PresenceState)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, err := res.LastInsertId()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p.ID = int(id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func updatePlayer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playerID := vars["id"]

	var p Player
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = db.Exec("UPDATE players SET name = ?, presence_date = ?, presence_status = ? WHERE id = ?", p.Name, p.PresenceDate, p.PresenceState, playerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func deletePlayer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playerID := vars["id"]

	_, err := db.Exec("DELETE FROM players WHERE id = ?", playerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func deleteAll(w http.ResponseWriter, r *http.Request) {
	_, err := db.Exec("DELETE FROM players")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

