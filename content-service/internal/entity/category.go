package entity

// Category entity
// ...existing code...

type Category struct {
	Base
	Name      string     `gorm:"type:varchar(100);not null;unique" json:"name"`
	Slug      string     `gorm:"type:varchar(150);not null;unique;index" json:"slug"`
	Showcases []Showcase `gorm:"foreignKey:CategoryID" json:"showcases,omitempty"`
}

// CategoryDTO untuk response tanpa deleted_at
// DTO ini digunakan di handler

type CategoryDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
