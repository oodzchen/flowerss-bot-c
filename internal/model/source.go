package model

type Source struct {
	ID         uint `gorm:"primary_key;AUTO_INCREMENT"`
	Link       string
	Title      string
	Language   string
	ErrorCount uint
	Content    []Content
	EditTime
}
