package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type SonarFile struct {
	ID              string    `gorm:"primaryKey;type:uuid" json:"id"`
	Name            string    `gorm:"type:varchar(255);not null" json:"name"`
	Path            string    `gorm:"type:varchar(512);not null" json:"path"`
	Width           int       `gorm:"not null" json:"width"`
	Height          int       `gorm:"not null" json:"height"`
	Size            int64     `gorm:"not null" json:"size"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	AnnotationCount int       `gorm:"default:0" json:"annotationCount"`
}

type Annotation struct {
	ID         string         `gorm:"primaryKey;type:uuid" json:"id"`
	FileID     string         `gorm:"type:uuid;not null;index" json:"fileId"`
	Type       string         `gorm:"type:varchar(20);not null" json:"type"`
	Points     datatypes.JSON `gorm:"type:jsonb;not null" json:"points"`
	CategoryID string         `gorm:"type:uuid;not null" json:"categoryId"`
	Label      string         `gorm:"type:varchar(100)" json:"label"`
	Color      string         `gorm:"type:varchar(20)" json:"color"`
	CreatedBy  string         `gorm:"type:varchar(100);not null" json:"createdBy"`
	Confidence *float64       `json:"confidence,omitempty"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

type Category struct {
	ID          string    `gorm:"primaryKey;type:uuid" json:"id"`
	Name        string    `gorm:"type:varchar(100);not null;unique" json:"name"`
	Color       string    `gorm:"type:varchar(20);not null" json:"color"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Snapshot struct {
	ID          string         `gorm:"primaryKey;type:uuid" json:"id"`
	FileID      string         `gorm:"type:uuid;not null;index" json:"fileId"`
	Annotations datatypes.JSON `gorm:"type:jsonb;not null" json:"annotations"`
	CreatedBy   string         `gorm:"type:varchar(100);not null" json:"createdBy"`
	Message     string         `gorm:"type:varchar(255)" json:"message"`
	Version     int            `gorm:"not null;default:0" json:"version"`
	CreatedAt   time.Time      `json:"createdAt"`
}

type WSMessage struct {
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	UserID    string          `json:"userId"`
	Timestamp int64           `json:"timestamp"`
}

func (sf *SonarFile) BeforeCreate(tx *gorm.DB) error {
	if sf.ID == "" {
		sf.ID = uuid.New().String()
	}
	return nil
}

func (a *Annotation) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

func (c *Category) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

func (s *Snapshot) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

func (a *Annotation) GetPoints() ([]Point, error) {
	var points []Point
	err := json.Unmarshal(a.Points, &points)
	return points, err
}

func (a *Annotation) SetPoints(points []Point) error {
	data, err := json.Marshal(points)
	if err != nil {
		return err
	}
	a.Points = datatypes.JSON(data)
	return nil
}
