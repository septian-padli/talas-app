package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/septianpadli/talas/content-service/internal/config"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/septianpadli/talas/content-service/pkg/database"
	"gorm.io/gorm"
)

func main() {
	// Parse Flags
	reset := flag.Bool("reset", false, "Reset database (Drop all tables & Re-migrate) before seeding")
	flag.Parse()

	// 1. Load Config
	cfg := config.LoadConfig()

	// 2. Ensure Database Exists
	ensureDatabase(cfg)

	// 3. Connect DB
	db := database.ConnectDB(cfg)

	// 4. Reset DB if flag is set
	if *reset {
		resetStats(db)
	}

	// 5. Seed Categories
	seedCategories(db)
}

// ensureDatabase checks and creates the database if it does not exist
func ensureDatabase(cfg *config.Config) {
	host := cfg.DBHost
	port := cfg.DBPort
	user := cfg.DBUser
	password := cfg.DBPassword
	dbName := cfg.DBName
	sslmode := cfg.DBSSLMode

	// Connect to default database (postgres)
	baseDSN := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s", host, port, user, password, sslmode)
	db, err := sql.Open("postgres", baseDSN)
	if err != nil {
		log.Fatalf("❌ Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	log.Printf("🔍 Checking if database '%s' exists...", dbName)
	var exists bool
	err = db.QueryRow("SELECT 1 FROM pg_database WHERE datname = $1", dbName).Scan(&exists)
	if err == sql.ErrNoRows || !exists {
		log.Printf("⚠️ Database '%s' does not exist. Creating...", dbName)
		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
		if err != nil {
			log.Fatalf("❌ Failed to create database '%s': %v", dbName, err)
		}
		log.Printf("✅ Database '%s' created successfully!", dbName)
	} else if err != nil && err != sql.ErrNoRows {
		log.Fatalf("❌ Error checking database existence: %v", err)
	} else {
		log.Printf("✅ Database '%s' already exists.", dbName)
	}
}

func resetStats(db *gorm.DB) {
	log.Println("🔥 Resetting Database (Dropping all tables)...")

	// 1. Drop Legacy/Orphaned Tables (Explicitly)
	legacyTables := []string{"project_likes", "project_media", "projects"}
	for _, tbl := range legacyTables {
		if err := db.Migrator().DropTable(tbl); err != nil {
			log.Printf("⚠️ Failed to drop legacy table %s: %v", tbl, err)
		} else {
			log.Printf("🗑️ Legacy table dropped: %s", tbl)
		}
	}

	// 2. Drop tables in compatible order (Foreign Keys)
	err := db.Migrator().DropTable(
		&entity.Collaborator{},
		&entity.Bookmark{},
		&entity.CommentLike{},
		&entity.ShowcaseLike{},
		&entity.Comment{},
		&entity.ShowcaseMedia{},
		&entity.Showcase{},
		&entity.Category{},
	)
	if err != nil {
		log.Fatalf("❌ Failed to drop tables: %v", err)
	}

	log.Println("🔄 Re-migrating tables...")
	err = db.AutoMigrate(
		&entity.Category{},
		&entity.Showcase{},
		&entity.ShowcaseMedia{},
		&entity.Comment{},
		&entity.ShowcaseLike{},
		&entity.CommentLike{},
		&entity.Bookmark{},
		&entity.Collaborator{},
	)
	if err != nil {
		log.Fatalf("❌ Failed to migrate tables: %v", err)
	}
	log.Println("✅ Database reset successful!")
}

func seedCategories(db *gorm.DB) {
	categories := []entity.Category{
		{Name: "Mobile Development", Slug: "mobile-development"},
		{Name: "Web Development", Slug: "web-development"},
		{Name: "Backend Engineering", Slug: "backend-engineering"},
		{Name: "UI/UX Design", Slug: "ui-ux-design"},
		{Name: "DevOps & Infrastructure", Slug: "devops-infrastructure"},
		{Name: "Data Science & AI", Slug: "data-science-ai"},
		{Name: "Game Development", Slug: "game-development"},
		{Name: "Cybersecurity", Slug: "cybersecurity"},
		{Name: "Blockchain & Web3", Slug: "blockchain-web3"},
	}

	log.Println("🌱 Seeding Categories...")

	for _, cat := range categories {
		// Use FirstOrCreate to avoid duplicates based on Slug
		if err := db.Where(entity.Category{Slug: cat.Slug}).FirstOrCreate(&cat).Error; err != nil {
			log.Printf("❌ Failed to seed category %s: %v\n", cat.Name, err)
		} else {
			log.Printf("✅ Category seeded: %s\n", cat.Name)
		}
	}

	log.Println("✨ Seeding completed successfully!")
}
