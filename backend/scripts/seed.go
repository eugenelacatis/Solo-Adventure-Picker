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
		{Name: "El Corte de Madera Creek Preserve", Type: "hike", Region: "bay-area", XPValue: 150, Lat: 37.3489, Lng: -122.2872, Description: "Wind through redwood groves and fern-lined singletrack on the Peninsula's favorite mountain-biking-and-hiking preserve."},
		{Name: "Mount Tamalpais", Type: "hike", Region: "bay-area", XPValue: 150, Lat: 37.9235, Lng: -122.5965, Description: "Climb the Bay Area's iconic peak for sweeping views across the Golden Gate, the ocean, and the whole North Bay."},
		{Name: "Sunol Regional Wilderness", Type: "hike", Region: "bay-area", XPValue: 150, Lat: 37.5136, Lng: -121.8310, Description: "Explore rolling oak-studded hills and the geologic curiosity known as Little Yosemite along Alameda Creek."},
		{Name: "Windy Hill Open Space Preserve", Type: "hike", Region: "bay-area", XPValue: 150, Lat: 37.3639, Lng: -122.2358, Description: "Grassy ridgelines with panoramic views of both the Santa Cruz Mountains and the South Bay below."},
		{Name: "Castle Rock State Park", Type: "hike", Region: "bay-area", XPValue: 150, Lat: 37.2280, Lng: -122.0928, Description: "Sandstone formations, a seasonal waterfall, and quiet forest trails deep in the Santa Cruz Mountains."},
		{Name: "Purisima Creek Redwoods", Type: "hike", Region: "bay-area", XPValue: 150, Lat: 37.3853, Lng: -122.3667, Description: "Descend through old-growth redwoods and creekside canyons on the way toward the coast."},
		{Name: "Lake Chabot Loop", Type: "hike", Region: "bay-area", XPValue: 150, Lat: 37.7183, Lng: -122.1122, Description: "An easygoing lakeside loop through the East Bay hills, good for a relaxed afternoon out."},
		{Name: "Edgewood Park and Natural Preserve", Type: "hike", Region: "bay-area", XPValue: 150, Lat: 37.4602, Lng: -122.2731, Description: "One of the Peninsula's best-known wildflower preserves, with serpentine grasslands and oak woodlands."},
		{Name: "Monte Bello Open Space Preserve", Type: "hike", Region: "bay-area", XPValue: 150, Lat: 37.3208, Lng: -122.1611, Description: "Ridge trails above Silicon Valley with views stretching from the bay to the Pacific on a clear day."},
		{Name: "Big Basin Redwoods", Type: "hike", Region: "bay-area", XPValue: 150, Lat: 37.1719, Lng: -122.2245, Description: "California's oldest state park, home to towering old-growth coast redwoods and fern canyons."},
		{Name: "Skyline Wilderness Park", Type: "hike", Region: "north-bay", XPValue: 150, Lat: 38.2694, Lng: -122.2653, Description: "A quiet Napa foothills park with rolling grassland trails and views over the valley."},
		{Name: "Point Reyes National Seashore", Type: "hike", Region: "north-bay", XPValue: 150, Lat: 38.0700, Lng: -122.9600, Description: "Dramatic coastal bluffs, windswept beaches, and a historic lighthouse at the edge of the continent."},
		{Name: "Annadel State Park", Type: "hike", Region: "north-bay", XPValue: 150, Lat: 38.4419, Lng: -122.6142, Description: "Volcanic rock outcrops and oak woodlands crisscrossed by trails just outside Santa Rosa."},
		{Name: "Almaden Quicksilver County Park", Type: "hike", Region: "south-bay", XPValue: 150, Lat: 37.1897, Lng: -121.8244, Description: "Trails through the remains of a 19th-century mercury mining district, now reclaimed by rolling hills."},
		{Name: "Rancho San Antonio Preserve", Type: "hike", Region: "south-bay", XPValue: 150, Lat: 37.3327, Lng: -122.0866, Description: "A South Bay favorite with oak woodlands, a working farm, and trails for every fitness level."},
		{Name: "Uvas Canyon County Park", Type: "hike", Region: "south-bay", XPValue: 150, Lat: 37.0850, Lng: -121.7736, Description: "A tucked-away canyon with a chain of small waterfalls, especially lively after winter rains."},
		{Name: "Redwood Regional Park", Type: "hike", Region: "east-bay", XPValue: 150, Lat: 37.8158, Lng: -122.1622, Description: "Second-growth redwoods towering over the Oakland hills, with quiet canyon trails below."},
		{Name: "Mission Peak Regional Preserve", Type: "hike", Region: "east-bay", XPValue: 150, Lat: 37.5091, Lng: -121.8807, Description: "A steep climb to the East Bay's most famous summit marker, with views across the whole valley."},
		{Name: "Diablo Foothills Regional Park", Type: "hike", Region: "east-bay", XPValue: 150, Lat: 37.8564, Lng: -121.9350, Description: "Rock formations and open grassland trails at the base of Mount Diablo."},
	}

	stmt, err := db.Prepare(`INSERT INTO adventures (name, type, region, description, xp_value, lat, lng) VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for _, adv := range seed {
		if _, err := stmt.Exec(adv.Name, adv.Type, adv.Region, adv.Description, adv.XPValue, adv.Lat, adv.Lng); err != nil {
			log.Fatal(err)
		}
	}
	log.Printf("Seed inserted with %d adventures!\n", len(seed))
}
