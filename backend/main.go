package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/eugenelacatis/solo-adventure-picker/config"
	"github.com/eugenelacatis/solo-adventure-picker/routes"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "solo-adventure-picker.db"
	}
	db := config.InitDB(dbPath)
	defer db.Close()

	mux := http.NewServeMux()
	routes.RegisterRoutes(mux, db)

	fmt.Println("Server running at http://localhost:8080")
	http.ListenAndServe(":8080", mux)
}
