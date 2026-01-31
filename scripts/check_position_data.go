package main

import (
	"fmt"
	"log"
	"nofx/config"
	"nofx/store"
)

// checkPositionData checks if position data exists for a specific trader
func main() {
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

	traderID := "72d7993b_5e8075de-237a-4b12-836e-ace7db0e39db_gemini_1767466138"
	db := st.GormDB()

	fmt.Printf("🔍 Checking position data for trader: %s\n\n", traderID)

	// Check total positions
	var totalCount int64
	db.Raw(`SELECT COUNT(*) FROM trader_positions WHERE trader_id = ?`, traderID).Scan(&totalCount)
	fmt.Printf("📊 Total positions: %d\n", totalCount)

	// Check CLOSED positions
	var closedCount int64
	db.Raw(`SELECT COUNT(*) FROM trader_positions WHERE trader_id = ? AND status = 'CLOSED'`, traderID).Scan(&closedCount)
	fmt.Printf("📊 CLOSED positions: %d\n", closedCount)

	// Check OPEN positions
	var openCount int64
	db.Raw(`SELECT COUNT(*) FROM trader_positions WHERE trader_id = ? AND status = 'OPEN'`, traderID).Scan(&openCount)
	fmt.Printf("📊 OPEN positions: %d\n", openCount)

	// Get sample CLOSED positions
	type PositionSample struct {
		ID          int64   `gorm:"column:id"`
		Symbol      string  `gorm:"column:symbol"`
		Side        string  `gorm:"column:side"`
		Status      string  `gorm:"column:status"`
		EntryTime   int64   `gorm:"column:entry_time"`
		ExitTime    int64   `gorm:"column:exit_time"`
		RealizedPnL float64 `gorm:"column:realized_pnl"`
	}
	var samples []PositionSample
	db.Raw(`
		SELECT id, symbol, side, status, entry_time, exit_time, realized_pnl 
		FROM trader_positions 
		WHERE trader_id = ? AND status = 'CLOSED' 
		ORDER BY exit_time DESC 
		LIMIT 5
	`, traderID).Scan(&samples)

	if len(samples) > 0 {
		fmt.Println("\n📋 Sample CLOSED positions (top 5):")
		for i, s := range samples {
			fmt.Printf("  %d. ID=%d, Symbol=%s, Side=%s, EntryTime=%d, ExitTime=%d, PnL=%.2f\n",
				i+1, s.ID, s.Symbol, s.Side, s.EntryTime, s.ExitTime, s.RealizedPnL)
		}
	} else {
		fmt.Println("\n⚠️ No CLOSED positions found!")
	}

	// Test GetClosedPositions function
	fmt.Println("\n🧪 Testing GetClosedPositions function...")
	positions, err := st.Position().GetClosedPositions(traderID, 10)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Printf("✅ GetClosedPositions returned %d positions\n", len(positions))
		if len(positions) > 0 {
			fmt.Println("   First position:")
			p := positions[0]
			fmt.Printf("   - ID: %d\n", p.ID)
			fmt.Printf("   - Symbol: %s\n", p.Symbol)
			fmt.Printf("   - Side: %s\n", p.Side)
			fmt.Printf("   - EntryTime: %d\n", p.EntryTime)
			fmt.Printf("   - ExitTime: %d\n", p.ExitTime)
			fmt.Printf("   - RealizedPnL: %.2f\n", p.RealizedPnL)
		}
	}

	// Check for positions with invalid status
	var invalidStatusCount int64
	db.Raw(`SELECT COUNT(*) FROM trader_positions WHERE trader_id = ? AND status NOT IN ('OPEN', 'CLOSED')`, traderID).Scan(&invalidStatusCount)
	if invalidStatusCount > 0 {
		fmt.Printf("\n⚠️ Found %d positions with invalid status\n", invalidStatusCount)
	}

	// Check for positions with exit_time = 0 but status = CLOSED
	var closedNoExitTime int64
	db.Raw(`SELECT COUNT(*) FROM trader_positions WHERE trader_id = ? AND status = 'CLOSED' AND exit_time = 0`, traderID).Scan(&closedNoExitTime)
	if closedNoExitTime > 0 {
		fmt.Printf("⚠️ Found %d CLOSED positions with exit_time = 0\n", closedNoExitTime)
	}
}
