package models

import (
	"time"
	"gorm.io/gorm"
)

// пользователь
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"unique;not null" json:"username"`
	Password  string         `gorm:"not null" json:"-"`
	Role      string         `gorm:"type:varchar(20);default:'reader'" json:"role"` // "admin", "editor", "reader"
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// статья
type Article struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Title     string         `gorm:"not null" json:"title"`
	AuthorID  uint           `json:"author_id"`
	Author    User           `json:"author"`
	Blocks    []Block        `gorm:"foreignKey:ArticleID;constraint:OnDelete:CASCADE" json:"blocks"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// блок статьи (заголовок + тело)
type Block struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	ArticleID uint    `json:"article_id"`
	Order     int     `json:"order"` // порядок блока в статье
	Heading   string  `json:"heading"`
	Body      string  `gorm:"type:text" json:"body"` // здесь может быть текст с маркдауном
	Images    []Image `gorm:"foreignKey:BlockID;constraint:OnDelete:CASCADE" json:"images"`
}

// картинка в блоке (с подписью)
type Image struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	BlockID uint   `json:"block_id"`
	URL     string `gorm:"not null" json:"url"`
	Caption string `json:"caption"`
}