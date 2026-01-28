package main

import (
	"flag"
	"log"

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

	// 2. Connect DB
	db := database.ConnectDB(cfg)

	// 3. Reset DB if flag is set
	if *reset {
		resetStats(db)
	}

	// 4. Seed Categories
	seedCategories(db)
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
