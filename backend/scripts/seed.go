package main

import (
	"log"
	"os"

	"github.com/eugenelacatis/solo-adventure-picker/config"
	"github.com/eugenelacatis/solo-adventure-picker/models"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "solo-adventure-picker.db"
	}
	db := config.InitDB(dbPath)
	defer db.Close()

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM adventures`).Scan(&n); err != nil {
		log.Println("Seed count error: ", err)
		return
	}

	if n > 0 {
		log.Printf("Adventures already seeded: %d\n", n)
		return
	}

	seed := []models.Adventure{
		{Name: "El Corte de Madera Creek Preserve", Type: "hike", Region: "bay-area", Lat: 37.3489, Lng: -122.2872},
		{Name: "Mount Tamalpais", Type: "hike", Region: "bay-area", Lat: 37.9235, Lng: -122.5965},
		{Name: "Sunol Regional Wilderness", Type: "hike", Region: "bay-area", Lat: 37.5136, Lng: -121.8310},
		{Name: "Windy Hill Open Space Preserve", Type: "hike", Region: "bay-area", Lat: 37.3639, Lng: -122.2358},
		{Name: "Castle Rock State Park", Type: "hike", Region: "bay-area", Lat: 37.2280, Lng: -122.0928},
		{Name: "Purisima Creek Redwoods", Type: "hike", Region: "bay-area", Lat: 37.3853, Lng: -122.3667},
		{Name: "Lake Chabot Loop", Type: "hike", Region: "bay-area", Lat: 37.7183, Lng: -122.1122},
		{Name: "Edgewood Park and Natural Preserve", Type: "hike", Region: "bay-area", Lat: 37.4602, Lng: -122.2731},
		{Name: "Monte Bello Open Space Preserve", Type: "hike", Region: "bay-area", Lat: 37.3208, Lng: -122.1611},
		{Name: "Big Basin Redwoods", Type: "hike", Region: "bay-area", Lat: 37.1719, Lng: -122.2245},
	}

	stmt, err := db.Prepare(`INSERT INTO adventures (name, type, region, lat, lng) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for _, adv := range seed {
		if _, err := stmt.Exec(adv.Name, adv.Type, adv.Region, adv.Lat, adv.Lng); err != nil {
			log.Fatal(err)
		}
	}
	log.Printf("Seed inserted with %d adventures!\n", len(seed))
}
