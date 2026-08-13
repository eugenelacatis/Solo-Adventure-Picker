package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/eugenelacatis/solo-adventure-picker/config"
	"github.com/eugenelacatis/solo-adventure-picker/routes"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	db := config.InitDB(databaseURL)
	defer db.Close()

	mux := http.NewServeMux()
	routes.RegisterRoutes(mux, db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server running at http://localhost:%s\n", port)
	http.ListenAndServe(":"+port, mux)
}
