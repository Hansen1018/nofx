package main

import (
	"fmt"
	"log"
	"nofx/config"
	"nofx/store"
	"time"
)

// migratePositionHistory migrates and fixes historical position data
func main() {
	// No need to initialize logger for migration script

	// Load config
	config.Init()
	cfg := config.Get()

	// Initialize database store
	var st *store.Store
	var err error
	if cfg.DBType == "postgres" {
		dbConfig := store.DBConfig{
			Type:     store.DBTypePostgres,
			Host:     cfg.DBHost,
			Port:     cfg.DBPort,
			User:     cfg.DBUser,
			Password: cfg.DBPassword,
			DBName:   cfg.DBName,
			SSLMode:  cfg.DBSSLMode,
		}
		st, err = store.NewWithConfig(dbConfig)
	} else {
		st, err = store.New(cfg.DBPath)
	}
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer st.Close()

	db := st.GormDB()

	fmt.Println("🔍 Starting position history migration...")

	// Step 1: Check for positions with missing or invalid status
	var countInvalidStatus int64
	// Use raw SQL for better compatibility
	db.Raw(`
		SELECT COUNT(*) FROM trader_positions 
		WHERE status NOT IN ('OPEN', 'CLOSED') OR status IS NULL
	`).Scan(&countInvalidStatus)

	if countInvalidStatus > 0 {
		fmt.Printf("📊 Found %d positions with invalid status, fixing...\n", countInvalidStatus)
		// Fix positions with invalid status: if exit_time > 0, mark as CLOSED, otherwise OPEN
		result := db.Exec(`
			UPDATE trader_positions 
			SET status = CASE 
				WHEN exit_time > 0 THEN 'CLOSED'
				ELSE 'OPEN'
			END
			WHERE status NOT IN ('OPEN', 'CLOSED') OR status IS NULL
		`)
		if result.Error != nil {
			log.Printf("⚠️ Warning: Failed to fix invalid status: %v", result.Error)
		} else {
			fmt.Printf("✅ Fixed %d positions with invalid status\n", result.RowsAffected)
		}
	}

	// Step 2: Fix positions that should be CLOSED but have status = OPEN
	var countOpenButClosed int64
	db.Raw(`
		SELECT COUNT(*) FROM trader_positions 
		WHERE status = 'OPEN' AND exit_time > 0
	`).Scan(&countOpenButClosed)

	if countOpenButClosed > 0 {
		fmt.Printf("📊 Found %d positions marked OPEN but have exit_time, fixing...\n", countOpenButClosed)
		result := db.Exec(`
			UPDATE trader_positions 
			SET status = 'CLOSED'
			WHERE status = 'OPEN' AND exit_time > 0
		`)
		if result.Error != nil {
			log.Printf("⚠️ Warning: Failed to fix OPEN positions with exit_time: %v", result.Error)
		} else {
			fmt.Printf("✅ Fixed %d positions marked OPEN but should be CLOSED\n", result.RowsAffected)
		}
	}

	// Step 3: Migrate timestamp formats (both SQLite and PostgreSQL)
	fmt.Println("📊 Checking timestamp formats...")
	
	if st.DBType() == store.DBTypeSQLite {
		// SQLite: Use raw SQL to check and convert timestamps
		// Check if there are any non-numeric timestamps
		var countStringTimestamps int64
		db.Raw(`
			SELECT COUNT(*) FROM trader_positions 
			WHERE typeof(entry_time) = 'text' 
			AND entry_time LIKE '%-%-%'
		`).Scan(&countStringTimestamps)
		
		if countStringTimestamps > 0 {
			fmt.Printf("🔄 Found %d positions with string timestamps, migrating using Go time.Parse...\n", countStringTimestamps)
			
			// Fetch all positions with string timestamps
			type PositionRow struct {
				ID        int64
				EntryTime string `gorm:"column:entry_time"`
				ExitTime  string `gorm:"column:exit_time"`
				CreatedAt string `gorm:"column:created_at"`
				UpdatedAt string `gorm:"column:updated_at"`
			}
			var positions []PositionRow
			db.Raw(`
				SELECT id, entry_time, exit_time, created_at, updated_at 
				FROM trader_positions 
				WHERE typeof(entry_time) = 'text' AND entry_time LIKE '%-%-%'
			`).Scan(&positions)
			
			fmt.Printf("   Processing %d positions...\n", len(positions))
			
			// Parse timestamp formats
			parseTimestamp := func(ts string) int64 {
				if ts == "" || ts == "0" {
					return 0
				}
				// Try multiple formats
				formats := []string{
					"2006-01-02 15:04:05.999999999+00:00",
					"2006-01-02 15:04:05.999999+00:00",
					"2006-01-02 15:04:05.999+00:00",
					"2006-01-02 15:04:05+00:00",
					"2006-01-02T15:04:05.999999999+00:00",
					"2006-01-02T15:04:05.999+00:00",
					"2006-01-02T15:04:05+00:00",
					"2006-01-02 15:04:05.999999999",
					"2006-01-02 15:04:05",
				}
				for _, format := range formats {
					if t, err := time.Parse(format, ts); err == nil {
						return t.UnixMilli()
					}
				}
				return 0
			}
			
			// Update each position
			updated := 0
			for _, pos := range positions {
				entryTimeMs := parseTimestamp(pos.EntryTime)
				exitTimeMs := parseTimestamp(pos.ExitTime)
				createdAtMs := parseTimestamp(pos.CreatedAt)
				updatedAtMs := parseTimestamp(pos.UpdatedAt)
				
				result := db.Exec(`
					UPDATE trader_positions 
					SET entry_time = ?, exit_time = ?, created_at = ?, updated_at = ?
					WHERE id = ?
				`, entryTimeMs, exitTimeMs, createdAtMs, updatedAtMs, pos.ID)
				
				if result.Error != nil {
					log.Printf("⚠️ Warning: Failed to update position %d: %v", pos.ID, result.Error)
				} else {
					updated++
				}
			}
			
			fmt.Printf("✅ Migrated %d positions\n", updated)
		} else {
			fmt.Println("✅ Timestamps appear to be in correct format (int64)")
		}
	} else {
		// PostgreSQL: convert timestamp to Unix milliseconds
		fmt.Println("🔄 Migrating PostgreSQL timestamps...")
		timestampColumns := []string{"entry_time", "exit_time", "created_at", "updated_at"}
		for _, col := range timestampColumns {
			result := db.Exec(fmt.Sprintf(`
				UPDATE trader_positions 
				SET %s = EXTRACT(EPOCH FROM %s::timestamp) * 1000
				WHERE %s IS NOT NULL 
				AND pg_typeof(%s)::text LIKE '%%timestamp%%'
			`, col, col, col, col))
			if result.Error != nil {
				log.Printf("⚠️ Warning: Failed to migrate %s: %v", col, result.Error)
			} else if result.RowsAffected > 0 {
				fmt.Printf("✅ Migrated %d %s values (PostgreSQL)\n", result.RowsAffected, col)
			}
		}
	}

	// Step 4: Fix positions with exit_time = 0 but should have exit_time
	// This handles cases where positions were closed but exit_time wasn't set
	var countClosedNoExitTime int64
	db.Raw(`
		SELECT COUNT(*) FROM trader_positions 
		WHERE status = 'CLOSED' AND exit_time = 0
	`).Scan(&countClosedNoExitTime)

	if countClosedNoExitTime > 0 {
		fmt.Printf("📊 Found %d CLOSED positions without exit_time, fixing...\n", countClosedNoExitTime)
		
		// First try: use updated_at if it's valid
		result1 := db.Exec(`
			UPDATE trader_positions 
			SET exit_time = updated_at
			WHERE status = 'CLOSED' AND exit_time = 0 AND updated_at > 0
		`)
		if result1.Error != nil {
			log.Printf("⚠️ Warning: Failed to set exit_time from updated_at: %v", result1.Error)
		} else if result1.RowsAffected > 0 {
			fmt.Printf("✅ Fixed %d positions using updated_at\n", result1.RowsAffected)
		}
		
		// Check remaining positions
		var remainingCount int64
		db.Raw(`SELECT COUNT(*) FROM trader_positions WHERE status = 'CLOSED' AND exit_time = 0`).Scan(&remainingCount)
		if remainingCount > 0 {
			fmt.Printf("📊 Still %d CLOSED positions without exit_time, using current time...\n", remainingCount)
			// Use current time as fallback
			nowMs := time.Now().UTC().UnixMilli()
			result2 := db.Exec(`
				UPDATE trader_positions 
				SET exit_time = ?
				WHERE status = 'CLOSED' AND exit_time = 0
			`, nowMs)
			if result2.Error != nil {
				log.Printf("⚠️ Warning: Failed to set exit_time to current time: %v", result2.Error)
			} else {
				fmt.Printf("✅ Fixed %d positions using current time\n", result2.RowsAffected)
			}
		}
	}

	// Step 5: Ensure entry_quantity is set for positions where it's 0
	var countMissingEntryQty int64
	db.Raw(`
		SELECT COUNT(*) FROM trader_positions 
		WHERE entry_quantity = 0 AND quantity > 0
	`).Scan(&countMissingEntryQty)

	if countMissingEntryQty > 0 {
		fmt.Printf("📊 Found %d positions with missing entry_quantity, fixing...\n", countMissingEntryQty)
		result := db.Exec(`
			UPDATE trader_positions 
			SET entry_quantity = quantity
			WHERE entry_quantity = 0 AND quantity > 0
		`)
		if result.Error != nil {
			log.Printf("⚠️ Warning: Failed to fix entry_quantity: %v", result.Error)
		} else {
			fmt.Printf("✅ Fixed %d positions with missing entry_quantity\n", result.RowsAffected)
		}
	}

	// Step 6: Statistics summary
	var totalClosed int64
	var totalOpen int64
	db.Raw(`SELECT COUNT(*) FROM trader_positions WHERE status = 'CLOSED'`).Scan(&totalClosed)
	db.Raw(`SELECT COUNT(*) FROM trader_positions WHERE status = 'OPEN'`).Scan(&totalOpen)

	fmt.Println("\n📈 Migration Summary:")
	fmt.Printf("  - Total CLOSED positions: %d\n", totalClosed)
	fmt.Printf("  - Total OPEN positions: %d\n", totalOpen)

	// Show per-trader breakdown
	type TraderCount struct {
		TraderID string `gorm:"column:trader_id"`
		Closed   int64  `gorm:"column:closed"`
		Open     int64  `gorm:"column:open"`
	}
	var traderCounts []TraderCount
	db.Raw(`
		SELECT 
			trader_id,
			SUM(CASE WHEN status = 'CLOSED' THEN 1 ELSE 0 END) as closed,
			SUM(CASE WHEN status = 'OPEN' THEN 1 ELSE 0 END) as open
		FROM trader_positions
		GROUP BY trader_id
		ORDER BY closed DESC
	`).Scan(&traderCounts)

	if len(traderCounts) > 0 {
		fmt.Println("\n📊 Per-trader breakdown:")
		for _, tc := range traderCounts {
			fmt.Printf("  - Trader %s: %d closed, %d open\n", tc.TraderID, tc.Closed, tc.Open)
		}
	}

	fmt.Println("\n✅ Position history migration completed!")
}
