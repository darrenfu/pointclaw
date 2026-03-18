package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/darrenfu/pointclaw/scraper/pkg/types"
)

// DB handles all Postgres operations for the scraper.
type DB struct {
	pool *pgxpool.Pool
}

// NewDB creates a new database connection pool.
func NewDB(ctx context.Context, databaseURL string) (*DB, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}

	// Disable prepared statement cache — required for PgBouncer/Supabase pooler
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{pool: pool}, nil
}

// Close closes the connection pool.
func (db *DB) Close() {
	db.pool.Close()
}

// CreateTables creates the schema if it doesn't exist.
func (db *DB) CreateTables(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS airports (
			code VARCHAR(3) PRIMARY KEY,
			name TEXT NOT NULL,
			city TEXT NOT NULL,
			country VARCHAR(2) NOT NULL,
			region TEXT NOT NULL,
			latitude DOUBLE PRECISION,
			longitude DOUBLE PRECISION,
			is_origin BOOLEAN DEFAULT FALSE
		)`,
		`CREATE TABLE IF NOT EXISTS routes (
			id SERIAL PRIMARY KEY,
			origin_code VARCHAR(3) NOT NULL REFERENCES airports(code),
			dest_code VARCHAR(3) NOT NULL REFERENCES airports(code),
			is_active BOOLEAN DEFAULT TRUE,
			UNIQUE(origin_code, dest_code)
		)`,
		`CREATE TABLE IF NOT EXISTS award_searches (
			id SERIAL PRIMARY KEY,
			origin_code VARCHAR(3) NOT NULL,
			dest_code VARCHAR(3) NOT NULL,
			search_date DATE NOT NULL,
			searched_at TIMESTAMPTZ DEFAULT NOW(),
			raw_response JSONB,
			status VARCHAR(20) DEFAULT 'success'
		)`,
		`CREATE INDEX IF NOT EXISTS idx_award_searches_route_date
			ON award_searches(origin_code, dest_code, search_date)`,
		`CREATE INDEX IF NOT EXISTS idx_award_searches_searched_at
			ON award_searches(searched_at)`,
		`CREATE TABLE IF NOT EXISTS award_flights (
			id SERIAL PRIMARY KEY,
			search_id INTEGER NOT NULL REFERENCES award_searches(id) ON DELETE CASCADE,
			flight_number TEXT NOT NULL,
			carrier_code VARCHAR(2) NOT NULL,
			carrier_name TEXT,
			origin VARCHAR(3) NOT NULL,
			destination VARCHAR(3) NOT NULL,
			departure_time TIMESTAMPTZ NOT NULL,
			arrival_time TIMESTAMPTZ NOT NULL,
			duration INTEGER NOT NULL,
			aircraft TEXT,
			cabin VARCHAR(20) NOT NULL,
			miles_cost INTEGER NOT NULL,
			cash_cost DOUBLE PRECISION NOT NULL,
			seats_remaining INTEGER,
			booking_code VARCHAR(5),
			is_saver_fare BOOLEAN DEFAULT FALSE,
			is_direct BOOLEAN DEFAULT TRUE,
			amenities JSONB
		)`,
		`CREATE INDEX IF NOT EXISTS idx_award_flights_search_id
			ON award_flights(search_id)`,
	}

	for _, q := range queries {
		if _, err := db.pool.Exec(ctx, q); err != nil {
			return fmt.Errorf("create table: %w\nquery: %s", err, q)
		}
	}
	slog.Info("database tables created/verified")
	return nil
}

// InsertSearch inserts an award search record and returns its ID.
func (db *DB) InsertSearch(ctx context.Context, origin, dest, date, status string, rawResponse json.RawMessage) (int, error) {
	var id int
	err := db.pool.QueryRow(ctx,
		`INSERT INTO award_searches (origin_code, dest_code, search_date, status, raw_response)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		origin, dest, date, status, string(rawResponse),
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert search: %w", err)
	}
	return id, nil
}

// InsertFlight inserts a normalized flight record.
func (db *DB) InsertFlight(ctx context.Context, searchID int, flight types.NormalizedFlight, fare types.NormalizedFare) error {
	amenitiesBytes, _ := json.Marshal(flight.Amenities)
	amenitiesStr := string(amenitiesBytes)

	_, err := db.pool.Exec(ctx,
		`INSERT INTO award_flights
		 (search_id, flight_number, carrier_code, carrier_name, origin, destination,
		  departure_time, arrival_time, duration, aircraft, cabin, miles_cost, cash_cost,
		  seats_remaining, booking_code, is_saver_fare, is_direct, amenities)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`,
		searchID, flight.FlightNumber, flight.Carrier.Code, flight.Carrier.Name,
		flight.Departure.Airport, flight.Arrival.Airport,
		parseTime(flight.Departure.Time), parseTime(flight.Arrival.Time),
		flight.Duration, flight.Aircraft,
		fare.Cabin, fare.Miles, fare.Cash, fare.SeatsRemaining,
		fare.BookingCode, fare.IsSaver, flight.IsDirect, amenitiesStr,
	)
	if err != nil {
		return fmt.Errorf("insert flight: %w", err)
	}
	return nil
}

// InsertSearchWithFlights inserts a search and all its flights in a transaction.
func (db *DB) InsertSearchWithFlights(ctx context.Context, origin, dest, date, status string, rawResponse json.RawMessage, flights []types.NormalizedFlight) (int, error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert search
	var searchID int
	err = tx.QueryRow(ctx,
		`INSERT INTO award_searches (origin_code, dest_code, search_date, status, raw_response)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		origin, dest, date, status, string(rawResponse),
	).Scan(&searchID)
	if err != nil {
		return 0, fmt.Errorf("insert search in tx: %w", err)
	}

	// Insert flights
	for _, flight := range flights {
		for _, fare := range flight.Fares {
			amenitiesBytes, _ := json.Marshal(flight.Amenities)
			amenitiesStr := string(amenitiesBytes) // string for simple protocol compatibility
			_, err = tx.Exec(ctx,
				`INSERT INTO award_flights
				 (search_id, flight_number, carrier_code, carrier_name, origin, destination,
				  departure_time, arrival_time, duration, aircraft, cabin, miles_cost, cash_cost,
				  seats_remaining, booking_code, is_saver_fare, is_direct, amenities)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`,
				searchID, flight.FlightNumber, flight.Carrier.Code, flight.Carrier.Name,
				flight.Departure.Airport, flight.Arrival.Airport,
				parseTime(flight.Departure.Time), parseTime(flight.Arrival.Time),
				flight.Duration, flight.Aircraft,
				fare.Cabin, fare.Miles, fare.Cash, fare.SeatsRemaining,
				fare.BookingCode, fare.IsSaver, flight.IsDirect, amenitiesStr,
			)
			if err != nil {
				return 0, fmt.Errorf("insert flight in tx: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}
	return searchID, nil
}

// GetRecentSearch checks if there's a fresh search result (within maxAge).
func (db *DB) GetRecentSearch(ctx context.Context, origin, dest, date string, maxAge time.Duration) (int, bool, error) {
	var id int
	err := db.pool.QueryRow(ctx,
		`SELECT id FROM award_searches
		 WHERE origin_code = $1 AND dest_code = $2 AND search_date = $3
		   AND searched_at > NOW() - $4::interval AND status = 'success'
		 ORDER BY searched_at DESC LIMIT 1`,
		origin, dest, date, fmt.Sprintf("%d seconds", int(maxAge.Seconds())),
	).Scan(&id)
	if err != nil {
		return 0, false, nil // no recent result
	}
	return id, true, nil
}

// GetFlightsBySearchID returns all flights for a given search.
func (db *DB) GetFlightsBySearchID(ctx context.Context, searchID int) ([]types.NormalizedFlight, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT flight_number, carrier_code, carrier_name, origin, destination,
		        departure_time, arrival_time, duration, aircraft,
		        cabin, miles_cost, cash_cost, seats_remaining, booking_code,
		        is_saver_fare, is_direct, amenities
		 FROM award_flights WHERE search_id = $1
		 ORDER BY miles_cost ASC`,
		searchID,
	)
	if err != nil {
		return nil, fmt.Errorf("query flights: %w", err)
	}
	defer rows.Close()

	// Group fares by flight number
	flightMap := make(map[string]*types.NormalizedFlight)
	var flightOrder []string

	for rows.Next() {
		var (
			flightNum, carrierCode, carrierName, origin, dest string
			depTime, arrTime                                  time.Time
			duration                                          int
			aircraft                                          *string
			cabin                                             string
			miles, seatsRemaining                             int
			cash                                              float64
			bookingCode                                       *string
			isSaver, isDirect                                 bool
			amenitiesJSON                                     []byte
		)
		err := rows.Scan(
			&flightNum, &carrierCode, &carrierName, &origin, &dest,
			&depTime, &arrTime, &duration, &aircraft,
			&cabin, &miles, &cash, &seatsRemaining, &bookingCode,
			&isSaver, &isDirect, &amenitiesJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("scan flight: %w", err)
		}

		fare := types.NormalizedFare{
			Cabin:          cabin,
			Miles:          miles,
			Cash:           cash,
			SeatsRemaining: seatsRemaining,
			IsSaver:        isSaver,
		}
		if bookingCode != nil {
			fare.BookingCode = *bookingCode
		}

		if f, ok := flightMap[flightNum]; ok {
			f.Fares = append(f.Fares, fare)
		} else {
			aircraftStr := ""
			if aircraft != nil {
				aircraftStr = *aircraft
			}
			var amenities []string
			json.Unmarshal(amenitiesJSON, &amenities)

			flight := &types.NormalizedFlight{
				FlightNumber: flightNum,
				Carrier:      types.CarrierInfo{Code: carrierCode, Name: carrierName},
				Departure:    types.AirportTime{Airport: origin, Time: depTime.Format(time.RFC3339)},
				Arrival:      types.AirportTime{Airport: dest, Time: arrTime.Format(time.RFC3339)},
				Duration:     duration,
				Aircraft:     aircraftStr,
				IsDirect:     isDirect,
				Fares:        []types.NormalizedFare{fare},
				Amenities:    amenities,
			}
			flightMap[flightNum] = flight
			flightOrder = append(flightOrder, flightNum)
		}
	}

	var flights []types.NormalizedFlight
	for _, fn := range flightOrder {
		flights = append(flights, *flightMap[fn])
	}
	return flights, nil
}

func parseTime(s string) time.Time {
	// Try RFC3339 first, then common formats
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
