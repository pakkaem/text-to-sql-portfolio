package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"time"

	_ "modernc.org/sqlite"
)

// Fallback schema untuk Smart City domain
const smartCitySchemaFallback = `
CREATE TABLE districts (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE, area_km2 REAL NOT NULL, population INTEGER NOT NULL, budget REAL DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE zones (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, district_id INTEGER NOT NULL REFERENCES districts(id), zone_type TEXT NOT NULL, area_km2 REAL NOT NULL);
CREATE TABLE roads (id INTEGER PRIMARY KEY AUTOINCREMENT, road_name TEXT NOT NULL, district_id INTEGER NOT NULL REFERENCES districts(id), zone_id INTEGER REFERENCES zones(id), road_type TEXT NOT NULL, lanes INTEGER DEFAULT 2, length_km REAL NOT NULL, speed_limit INTEGER DEFAULT 40);
CREATE TABLE camera_locations (id INTEGER PRIMARY KEY AUTOINCREMENT, road_id INTEGER NOT NULL REFERENCES roads(id), location_desc TEXT NOT NULL, camera_type TEXT NOT NULL, installed_date DATE NOT NULL, is_active BOOLEAN DEFAULT 1);
CREATE TABLE traffic_readings (id INTEGER PRIMARY KEY AUTOINCREMENT, camera_id INTEGER NOT NULL REFERENCES camera_locations(id), reading_time DATETIME NOT NULL, vehicle_count INTEGER NOT NULL, avg_speed REAL, congestion_level TEXT, is_peak_hour BOOLEAN DEFAULT 0);
CREATE TABLE violations (id INTEGER PRIMARY KEY AUTOINCREMENT, camera_id INTEGER NOT NULL REFERENCES camera_locations(id), vehicle_plate TEXT NOT NULL, violation_type TEXT NOT NULL, violation_time DATETIME NOT NULL, speed_recorded REAL, speed_limit INTEGER, fine_amount REAL NOT NULL, status TEXT DEFAULT 'pending');
CREATE TABLE weather_data (id INTEGER PRIMARY KEY AUTOINCREMENT, zone_id INTEGER NOT NULL REFERENCES zones(id), recorded_at DATETIME NOT NULL, data_type TEXT NOT NULL, temperature REAL, humidity REAL, wind_speed REAL, pm25 REAL, pm10 REAL, air_quality_index INTEGER, weather_condition TEXT);
CREATE TABLE incidents (id INTEGER PRIMARY KEY AUTOINCREMENT, district_id INTEGER NOT NULL REFERENCES districts(id), incident_type TEXT NOT NULL, severity TEXT NOT NULL, description TEXT, reported_at DATETIME NOT NULL, resolved_at DATETIME, status TEXT DEFAULT 'open', response_time_minutes INTEGER);
CREATE TABLE infrastructure_projects (id INTEGER PRIMARY KEY AUTOINCREMENT, project_name TEXT NOT NULL, district_id INTEGER NOT NULL REFERENCES districts(id), project_type TEXT NOT NULL, budget REAL NOT NULL, status TEXT DEFAULT 'planned', start_date DATE, end_date DATE);
`

func initSmartCityDB() *sql.DB {
	dbName := getEnv("SMARTCITY_DB_PATH", "./smartcity.db")
	db, err := sql.Open("sqlite", dbName)
	if err != nil {
		log.Fatal(err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS districts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		area_km2 REAL NOT NULL,
		population INTEGER NOT NULL,
		budget REAL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS zones (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		district_id INTEGER NOT NULL,
		zone_type TEXT NOT NULL CHECK(zone_type IN ('residential', 'commercial', 'industrial', 'public', 'green')),
		area_km2 REAL NOT NULL,
		FOREIGN KEY (district_id) REFERENCES districts(id)
	);

	CREATE TABLE IF NOT EXISTS roads (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		road_name TEXT NOT NULL,
		district_id INTEGER NOT NULL,
		zone_id INTEGER,
		road_type TEXT NOT NULL CHECK(road_type IN ('highway', 'arterial', 'collector', 'local', 'pedestrian')),
		lanes INTEGER DEFAULT 2,
		length_km REAL NOT NULL,
		speed_limit INTEGER DEFAULT 40,
		FOREIGN KEY (district_id) REFERENCES districts(id),
		FOREIGN KEY (zone_id) REFERENCES zones(id)
	);

	CREATE TABLE IF NOT EXISTS camera_locations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		road_id INTEGER NOT NULL,
		location_desc TEXT NOT NULL,
		camera_type TEXT NOT NULL CHECK(camera_type IN ('traffic', 'surveillance', 'speed', 'parking')),
		installed_date DATE NOT NULL,
		is_active BOOLEAN DEFAULT 1,
		FOREIGN KEY (road_id) REFERENCES roads(id)
	);

	CREATE TABLE IF NOT EXISTS traffic_readings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		camera_id INTEGER NOT NULL,
		reading_time DATETIME NOT NULL,
		vehicle_count INTEGER NOT NULL,
		avg_speed REAL,
		congestion_level TEXT CHECK(congestion_level IN ('low', 'medium', 'high', 'severe')),
		is_peak_hour BOOLEAN DEFAULT 0,
		FOREIGN KEY (camera_id) REFERENCES camera_locations(id)
	);

	CREATE TABLE IF NOT EXISTS violations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		camera_id INTEGER NOT NULL,
		vehicle_plate TEXT NOT NULL,
		violation_type TEXT NOT NULL CHECK(violation_type IN ('speeding', 'red_light', 'illegal_parking', 'wrong_way', 'no_helmet', 'expired_registration')),
		violation_time DATETIME NOT NULL,
		speed_recorded REAL,
		speed_limit INTEGER,
		fine_amount REAL NOT NULL,
		status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'paid', 'contested', 'dismissed')),
		FOREIGN KEY (camera_id) REFERENCES camera_locations(id)
	);

	CREATE TABLE IF NOT EXISTS weather_data (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		zone_id INTEGER NOT NULL,
		recorded_at DATETIME NOT NULL,
		data_type TEXT NOT NULL CHECK(data_type IN ('temperature', 'humidity', 'air_quality', 'rainfall')),
		temperature REAL,
		humidity REAL,
		wind_speed REAL,
		pm25 REAL,
		pm10 REAL,
		air_quality_index INTEGER,
		weather_condition TEXT CHECK(weather_condition IN ('clear', 'cloudy', 'rain', 'heavy_rain', 'fog', 'haze')),
		FOREIGN KEY (zone_id) REFERENCES zones(id)
	);

	CREATE TABLE IF NOT EXISTS incidents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		district_id INTEGER NOT NULL,
		incident_type TEXT NOT NULL CHECK(incident_type IN ('accident', 'fire', 'flood', 'power_outage', 'road_damage', 'crime')),
		severity TEXT NOT NULL CHECK(severity IN ('low', 'medium', 'high', 'critical')),
		description TEXT,
		reported_at DATETIME NOT NULL,
		resolved_at DATETIME,
		status TEXT DEFAULT 'open' CHECK(status IN ('open', 'in_progress', 'resolved', 'closed')),
		response_time_minutes INTEGER,
		FOREIGN KEY (district_id) REFERENCES districts(id)
	);

	CREATE TABLE IF NOT EXISTS infrastructure_projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_name TEXT NOT NULL,
		district_id INTEGER NOT NULL,
		project_type TEXT NOT NULL CHECK(project_type IN ('road', 'bridge', 'utility', 'park', 'building', 'water', 'telecom')),
		budget REAL NOT NULL,
		status TEXT DEFAULT 'planned' CHECK(status IN ('planned', 'in_progress', 'completed', 'on_hold', 'cancelled')),
		start_date DATE,
		end_date DATE,
		FOREIGN KEY (district_id) REFERENCES districts(id)
	);

	CREATE INDEX IF NOT EXISTS idx_zones_district ON zones(district_id);
	CREATE INDEX IF NOT EXISTS idx_roads_district ON roads(district_id);
	CREATE INDEX IF NOT EXISTS idx_roads_zone ON roads(zone_id);
	CREATE INDEX IF NOT EXISTS idx_cameras_road ON camera_locations(road_id);
	CREATE INDEX IF NOT EXISTS idx_traffic_camera ON traffic_readings(camera_id);
	CREATE INDEX IF NOT EXISTS idx_traffic_time ON traffic_readings(reading_time);
	CREATE INDEX IF NOT EXISTS idx_violations_camera ON violations(camera_id);
	CREATE INDEX IF NOT EXISTS idx_violations_time ON violations(violation_time);
	CREATE INDEX IF NOT EXISTS idx_weather_zone ON weather_data(zone_id);
	CREATE INDEX IF NOT EXISTS idx_incidents_district ON incidents(district_id);
	CREATE INDEX IF NOT EXISTS idx_infra_district ON infrastructure_projects(district_id);
	`
	_, err = db.Exec(schema)
	if err != nil {
		log.Fatal("Gagal membuat skema Smart City: ", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM districts").Scan(&count)

	if count == 0 {
		seedSmartCityData(db)
	} else {
		log.Println("Database Smart City sudah berisi data, siap digunakan.")
	}

	return db
}

func seedSmartCityData(db *sql.DB) {
	tx, err := db.Begin()
	if err != nil {
		log.Printf("WARNING: Gagal mulai transaksi seed Smart City: %v", err)
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	rng := rand.New(rand.NewSource(42))
	now := time.Now()

	// === DISTRICTS (8 districts) ===
	districts := []struct {
		name       string
		area       float64
		population int
		budget     float64
	}{
		{"Central Jakarta", 48.13, 912000, 2500000000},
		{"North Jakarta", 146.66, 1820000, 1800000000},
		{"South Jakarta", 145.73, 2250000, 2200000000},
		{"East Jakarta", 188.03, 2850000, 2000000000},
		{"West Jakarta", 124.44, 2430000, 1900000000},
		{"Bandung Kota", 167.67, 2500000, 1500000000},
		{"Surabaya Kota", 350.54, 2870000, 2100000000},
		{"Semarang Kota", 373.78, 1800000, 1200000000},
	}

	districtIDs := make([]int64, len(districts))
	for i, d := range districts {
		result, e := tx.Exec(
			"INSERT INTO districts (name, area_km2, population, budget) VALUES (?, ?, ?, ?)",
			d.name, d.area, d.population, d.budget,
		)
		if e != nil {
			err = fmt.Errorf("insert district %s: %w", d.name, e)
			return
		}
		districtIDs[i], _ = result.LastInsertId()
	}

	// === ZONES (4-6 per district) ===
	zoneTypes := []string{"residential", "commercial", "industrial", "public", "green"}
	zoneNames := []string{
		"Downtown", "Market Area", "Tech Park", "University District", "Harbor Zone",
		"Green Belt", "Mall Complex", "Industrial Estate", "Government Quarter", "Cultural Center",
		"Sports District", "Medical Hub", "Transportation Hub", "Residential Complex",
	}
	zoneIDs := make([]int64, 0)
	zoneDistrictMap := make(map[int64][]int64)

	for i, distID := range districtIDs {
		numZones := 4 + rng.Intn(3)
		for j := 0; j < numZones; j++ {
			name := fmt.Sprintf("%s %s", districts[i].name, zoneNames[(i*4+j)%len(zoneNames)])
			zType := zoneTypes[(i+j)%len(zoneTypes)]
			area := 0.5 + rng.Float64()*15.0
			result, e := tx.Exec(
				"INSERT INTO zones (name, district_id, zone_type, area_km2) VALUES (?, ?, ?, ?)",
				name, distID, zType, area,
			)
			if e != nil {
				err = fmt.Errorf("insert zone: %w", e)
				return
			}
			zid, _ := result.LastInsertId()
			zoneIDs = append(zoneIDs, zid)
			zoneDistrictMap[distID] = append(zoneDistrictMap[distID], zid)
		}
	}

	// === ROADS (5-8 per district) ===
	roadNames := []string{
		"Jl. Sudirman", "Jl. Thamrin", "Jl. Gatot Subroto", "Jl. Rasuna Said",
		"Jl. MH Thamrin", "Jl. Diponegoro", "Jl. Imam Bonjol", "Jl. Hayam Wuruk",
		"Jl. Gajah Mada", "Jl. Pahlawan", "Jl. Merdeka", "Jl. Asia Afrika",
		"Jl. Pemuda", "Jl. Veteran", "Jl. Ahmad Yani", "Jl. Sudirman Timur",
	}
	roadTypes := []string{"highway", "arterial", "collector", "local", "pedestrian"}
	roadIDs := make([]int64, 0)
	roadDistrictMap := make(map[int64][]int64)

	for _, distID := range districtIDs {
		numRoads := 5 + rng.Intn(4)
		for j := 0; j < numRoads; j++ {
			roadName := roadNames[(int(distID)*3+j)%len(roadNames)]
			if j > 3 {
				roadName = fmt.Sprintf("%s Extension %d", roadName, j)
			}
			rType := roadTypes[rng.Intn(len(roadTypes))]
			lanes := 2 + rng.Intn(5)
			length := 0.5 + rng.Float64()*20.0
			speedLimit := 30 + rng.Intn(5)*10

			dZones := zoneDistrictMap[distID]
			var result sql.Result
			var e error
			if len(dZones) > 0 {
				zid := dZones[rng.Intn(len(dZones))]
				result, e = tx.Exec(
					"INSERT INTO roads (road_name, district_id, zone_id, road_type, lanes, length_km, speed_limit) VALUES (?, ?, ?, ?, ?, ?, ?)",
					roadName, distID, zid, rType, lanes, length, speedLimit,
				)
			} else {
				result, e = tx.Exec(
					"INSERT INTO roads (road_name, district_id, zone_id, road_type, lanes, length_km, speed_limit) VALUES (?, ?, NULL, ?, ?, ?, ?)",
					roadName, distID, rType, lanes, length, speedLimit,
				)
			}
			if e != nil {
				err = fmt.Errorf("insert road: %w", e)
				return
			}
			rid, _ := result.LastInsertId()
			roadIDs = append(roadIDs, rid)
			roadDistrictMap[distID] = append(roadDistrictMap[distID], rid)
		}
	}

	// === CAMERA LOCATIONS (~100 total) ===
	cameraTypes := []string{"traffic", "surveillance", "speed", "parking"}
	locationDescs := []string{
		"Near intersection", "Highway overpass", "Toll gate entrance", "School zone",
		"Market entrance", "Bridge approach", "Tunnel entrance", "Roundabout",
		"Mall parking lot", "Bus stop", "Train station exit", "Hospital entrance",
	}
	cameraIDs := make([]int64, 0)

	for _, roadID := range roadIDs {
		numCameras := 2 + rng.Intn(3)
		for j := 0; j < numCameras; j++ {
			locDesc := locationDescs[rng.Intn(len(locationDescs))]
			cType := cameraTypes[rng.Intn(len(cameraTypes))]
			daysAgo := rng.Intn(730)
			installed := now.AddDate(0, 0, -daysAgo).Format("2006-01-02")
			isActive := rng.Float64() > 0.05

			result, e := tx.Exec(
				"INSERT INTO camera_locations (road_id, location_desc, camera_type, installed_date, is_active) VALUES (?, ?, ?, ?, ?)",
				roadID, locDesc, cType, installed, isActive,
			)
			if e != nil {
				err = fmt.Errorf("insert camera: %w", e)
				return
			}
			camID, _ := result.LastInsertId()
			cameraIDs = append(cameraIDs, camID)
		}
	}

	// === TRAFFIC READINGS (100 entries) ===
	congestionLevels := []string{"low", "medium", "high", "severe"}
	peakHours := map[int]bool{7: true, 8: true, 9: true, 17: true, 18: true, 19: true}

	for i := 0; i < 100; i++ {
		cameraID := cameraIDs[rng.Intn(len(cameraIDs))]
		daysAgo := rng.Intn(30)
		hour := rng.Intn(24)
		minute := rng.Intn(60)
		readingTime := now.AddDate(0, 0, -daysAgo)
		readingTime = time.Date(readingTime.Year(), readingTime.Month(), readingTime.Day(), hour, minute, 0, 0, time.UTC)

		vehicleCount := 10 + rng.Intn(500)
		if peakHours[hour] {
			vehicleCount = 200 + rng.Intn(400)
		}
		avgSpeed := 15.0 + rng.Float64()*60.0
		congestion := congestionLevels[rng.Intn(len(congestionLevels))]
		if vehicleCount > 400 {
			congestion = "severe"
		} else if vehicleCount > 300 {
			congestion = "high"
		}
		isPeak := peakHours[hour]

		_, e := tx.Exec(
			"INSERT INTO traffic_readings (camera_id, reading_time, vehicle_count, avg_speed, congestion_level, is_peak_hour) VALUES (?, ?, ?, ?, ?, ?)",
			cameraID, readingTime.Format("2006-01-02 15:04:05"), vehicleCount, avgSpeed, congestion, isPeak,
		)
		if e != nil {
			err = fmt.Errorf("insert traffic reading: %w", e)
			return
		}
	}

	// === VIOLATIONS (75 entries) ===
	violationTypes := []string{"speeding", "red_light", "illegal_parking", "wrong_way", "no_helmet", "expired_registration"}
	violationStatuses := []string{"pending", "paid", "contested", "dismissed"}
	plates := []string{"B 1234 CD", "D 5678 EF", "AB 9012 GH", "B 3456 IJ", "F 7890 KL",
		"AD 2345 MN", "B 6789 OP", "D 1234 QR", "AB 5678 ST", "B 9012 UV",
		"B 4567 WX", "H 8901 YZ", "L 2345 AB", "N 6789 CD", "B 0123 EF"}

	for i := 0; i < 75; i++ {
		cameraID := cameraIDs[rng.Intn(len(cameraIDs))]
		plate := plates[rng.Intn(len(plates))]
		vType := violationTypes[rng.Intn(len(violationTypes))]
		daysAgo := rng.Intn(90)
		hour := rng.Intn(24)
		vTime := now.AddDate(0, 0, -daysAgo)
		vTime = time.Date(vTime.Year(), vTime.Month(), vTime.Day(), hour, rng.Intn(60), rng.Intn(60), 0, time.UTC)

		var speedRecorded *float64
		var speedLimit *int
		var fineAmount float64

		switch vType {
		case "speeding":
			s := 60.0 + rng.Float64()*80.0
			speedRecorded = &s
			l := 40 + rng.Intn(4)*10
			speedLimit = &l
			fineAmount = 500000 + float64(rng.Intn(10))*100000
		case "red_light":
			fineAmount = 500000
		case "illegal_parking":
			fineAmount = 250000
		case "wrong_way":
			fineAmount = 750000
		case "no_helmet":
			fineAmount = 250000
		case "expired_registration":
			fineAmount = 500000
		}

		status := violationStatuses[rng.Intn(len(violationStatuses))]

		_, e := tx.Exec(
			`INSERT INTO violations (camera_id, vehicle_plate, violation_type, violation_time, speed_recorded, speed_limit, fine_amount, status) 
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			cameraID, plate, vType, vTime.Format("2006-01-02 15:04:05"), speedRecorded, speedLimit, fineAmount, status,
		)
		if e != nil {
			err = fmt.Errorf("insert violation: %w", e)
			return
		}
	}

	// === WEATHER DATA (60 entries) ===
	weatherConditions := []string{"clear", "cloudy", "rain", "heavy_rain", "fog", "haze"}
	weatherTypes := []string{"temperature", "humidity", "air_quality", "rainfall"}

	for i := 0; i < 60; i++ {
		zoneID := zoneIDs[rng.Intn(len(zoneIDs))]
		daysAgo := rng.Intn(30)
		hour := rng.Intn(24)
		recordedAt := now.AddDate(0, 0, -daysAgo)
		recordedAt = time.Date(recordedAt.Year(), recordedAt.Month(), recordedAt.Day(), hour, rng.Intn(60), 0, 0, time.UTC)

		dType := weatherTypes[rng.Intn(len(weatherTypes))]
		temp := 22.0 + rng.Float64()*14.0
		humidity := 40.0 + rng.Float64()*55.0
		windSpeed := rng.Float64() * 30.0
		pm25 := 5.0 + rng.Float64()*150.0
		pm10 := pm25 + rng.Float64()*50.0
		aqi := int(pm25 * 0.8)
		condition := weatherConditions[rng.Intn(len(weatherConditions))]

		_, e := tx.Exec(
			`INSERT INTO weather_data (zone_id, recorded_at, data_type, temperature, humidity, wind_speed, pm25, pm10, air_quality_index, weather_condition)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			zoneID, recordedAt.Format("2006-01-02 15:04:05"), dType, temp, humidity, windSpeed, pm25, pm10, aqi, condition,
		)
		if e != nil {
			err = fmt.Errorf("insert weather: %w", e)
			return
		}
	}

	// === INCIDENTS (40 entries) ===
	incidentTypes := []string{"accident", "fire", "flood", "power_outage", "road_damage", "crime"}
	severities := []string{"low", "medium", "high", "critical"}
	incidentStatuses := []string{"open", "in_progress", "resolved", "closed"}

	for i := 0; i < 40; i++ {
		distID := districtIDs[rng.Intn(len(districtIDs))]
		iType := incidentTypes[rng.Intn(len(incidentTypes))]
		severity := severities[rng.Intn(len(severities))]
		daysAgo := rng.Intn(180)
		reportedAt := now.AddDate(0, 0, -daysAgo)
		hour := rng.Intn(24)
		reportedAt = time.Date(reportedAt.Year(), reportedAt.Month(), reportedAt.Day(), hour, rng.Intn(60), 0, 0, time.UTC)

		var resolvedAt *string
		var responseTime *int
		status := incidentStatuses[rng.Intn(len(incidentStatuses))]

		if status == "resolved" || status == "closed" {
			resolveOffset := 30 + rng.Intn(1440)
			rAt := reportedAt.Add(time.Duration(resolveOffset) * time.Minute)
			rAtStr := rAt.Format("2006-01-02 15:04:05")
			resolvedAt = &rAtStr
			responseTime = &resolveOffset
		}

		descriptions := []string{
			fmt.Sprintf("%s reported in district area", iType),
			fmt.Sprintf("Multiple vehicles involved in %s", iType),
			fmt.Sprintf("Minor %s under investigation", iType),
			fmt.Sprintf("Emergency response dispatched for %s", iType),
		}
		desc := descriptions[rng.Intn(len(descriptions))]

		_, e := tx.Exec(
			`INSERT INTO incidents (district_id, incident_type, severity, description, reported_at, resolved_at, status, response_time_minutes)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			distID, iType, severity, desc, reportedAt.Format("2006-01-02 15:04:05"), resolvedAt, status, responseTime,
		)
		if e != nil {
			err = fmt.Errorf("insert incident: %w", e)
			return
		}
	}

	// === INFRASTRUCTURE PROJECTS (30 entries) ===
	projectNames := []string{
		"Road Widening", "Bridge Construction", "Water Treatment Plant", "Smart Traffic Light",
		"Fiber Optic Network", "Public Park Renovation", "Flood Canal Dredging", "Electric Bus Terminal",
		"Street Lighting Upgrade", "Waste Management Center", "Pedestrian Overpass", "Storm Drain System",
		"Renewable Energy Station", "Community Health Center", "Smart Parking Facility",
	}
	projectTypes := []string{"road", "bridge", "utility", "park", "building", "water", "telecom"}
	projectStatuses := []string{"planned", "in_progress", "completed", "on_hold", "cancelled"}

	for i := 0; i < 30; i++ {
		distID := districtIDs[rng.Intn(len(districtIDs))]
		pName := projectNames[rng.Intn(len(projectNames))]
		pType := projectTypes[rng.Intn(len(projectTypes))]
		budget := 500000000 + rng.Float64()*5000000000
		status := projectStatuses[rng.Intn(len(projectStatuses))]

		startDaysAgo := rng.Intn(365)
		startDate := now.AddDate(0, 0, -startDaysAgo).Format("2006-01-02")
		durationDays := 60 + rng.Intn(540)
		endDate := now.AddDate(0, 0, -startDaysAgo+durationDays).Format("2006-01-02")

		_, e := tx.Exec(
			`INSERT INTO infrastructure_projects (project_name, district_id, project_type, budget, status, start_date, end_date)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			pName, distID, pType, budget, status, startDate, endDate,
		)
		if e != nil {
			err = fmt.Errorf("insert infrastructure project: %w", e)
			return
		}
	}

	if e := tx.Commit(); e != nil {
		log.Printf("WARNING: Gagal commit seed Smart City: %v", e)
		return
	}

	log.Println("Database Smart City berhasil diinisialisasi beserta data dummy!")
}