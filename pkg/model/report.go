package model

import (
	"html/template"
	"time"

	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
	"gorm.io/gorm"
)

type ReportState string

const (
	ReportStateOpen ReportState = "open"
	ReportStateDone ReportState = "done"
)

type Report struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time

	ReporterToken      string `gorm:"type:varchar(255);not null"`
	AdministratorToken string `gorm:"type:varchar(255);not null"`

	Messages []Message `gorm:"foreignKey:ReportID;constraint:OnDelete:CASCADE"`

	State ReportState
}

func (r *Report) GetStatusColor() template.CSS {
	switch r.State {
	case ReportStateOpen:
		return "rgb(220 38 38)"
	case ReportStateDone:
		return "rgb(74 222 128)"
	default:
		return "black"
	}
}

func (r *Report) GetStatusText() string {
	switch r.State {
	case ReportStateOpen:
		return "Open"
	case ReportStateDone:
		return "Done"
	default:
		return "Unknown"
	}
}

type Message struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time

	Content string `gorm:"not null"`

	ReportID uint `gorm:"not null"`

	Files   []File `gorm:"many2many:message_files;"`
	IsAdmin bool   `gorm:"default:false"`
}

type File struct {
	ID       uint   `gorm:"primarykey"`
	UUID     string `gorm:"unique;not null"`
	Location string `gorm:"not null"`
	Name     string `gorm:"not null"`
}

func (f *File) BeforeCreate(tx *gorm.DB) (err error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	f.UUID = id.String()
	return nil
}

func (r *Report) BeforeCreate(tx *gorm.DB) (err error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	r.ReporterToken = id.String()
	id, err = uuid.NewRandom()
	if err != nil {
		return err
	}
	r.AdministratorToken = id.String()
	return nil
}

func (m *Message) GetBody() template.HTML {
	html := blackfriday.Run([]byte(m.Content), blackfriday.WithExtensions(blackfriday.CommonExtensions|blackfriday.HardLineBreak))
	p := bluemonday.NewPolicy()
	p.AllowStandardURLs()
	p.AllowAttrs("href").OnElements("a")
	p.AllowElements("b", "br", "strong", "p", "ul", "li")
	sanitized := p.Sanitize(string(html))
	return template.HTML(sanitized)
}
