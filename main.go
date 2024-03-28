package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

type Todo struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	IsCompleted bool   `json:"isCompleted"`
}

var templates map[string]*template.Template

func init() {
	if templates == nil {
		templates = make(map[string]*template.Template)
	}

	templates["index.html"] = template.Must(template.ParseFiles("index.html"))
	templates["todo.html"] = template.Must(template.ParseFiles("todo.html"))
}

func initDB(db *sql.DB) error {
	const createTableSQL = `CREATE TABLE IF NOT EXISTS todos (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        isCompleted BOOLEAN NOT NULL DEFAULT 0
    );`
	_, err := db.Exec(createTableSQL)
	return err
}

// handlers
func submitTodoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PostFormValue("name")
		completed := r.PostFormValue("completed") == "true"

		var lastInsertId int
		err := db.QueryRow("INSERT INTO todos (name, isCompleted) VALUES ($1, $2) RETURNING id", name, completed).Scan(&lastInsertId)
		if err != nil {
			log.Fatal(err)
		}

		todo := Todo{Id: lastInsertId, Name: name, IsCompleted: completed}

		tmpl := templates["todo.html"]
		tmpl.ExecuteTemplate(w, "todo.html", todo)
	}
}

func deleteTodoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.FormValue("id")

		fmt.Println("id = ", id)
		_, err := db.Exec("DELETE FROM todos WHERE id = ?", id)
		if err != nil {
			http.Error(w, "Failed to delete todo", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func indexHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var todos []Todo
		rows, err := db.Query("SELECT id, name, isCompleted FROM todos")
		if err != nil {
			log.Fatal("Db querry error", err)
		}
		defer rows.Close()

		for rows.Next() {
			var todo Todo
			err := rows.Scan(&todo.Id, &todo.Name, &todo.IsCompleted)
			if err != nil {
				log.Fatal("Db scan error", err)
			}
			todos = append(todos, todo)
		}
		if err = rows.Err(); err != nil {
			log.Fatal(err)
		}

		json, err := json.Marshal(todos)
		if err != nil {
			log.Fatal(err)
		}

		tmpl := templates["index.html"]
		tmpl.ExecuteTemplate(w, "index.html", map[string]template.JS{"Todos": template.JS(json)})
	}
}

func main() {
	db, err := sql.Open("sqlite3", "db.sqlite")
	if err != nil {
		log.Fatal("Failed to open database", err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	http.HandleFunc("/", indexHandler(db))
	http.HandleFunc("/submit-todo/", submitTodoHandler(db))
	http.HandleFunc("/delete-todo/", deleteTodoHandler(db))
	log.Fatal(http.ListenAndServe(":8000", nil))
}
