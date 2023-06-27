package model

import (
	"html"
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
	ReportStateSpam ReportState = "spam"
)

type Report struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	TopicID   uint

	ReporterToken      string `gorm:"type:varchar(255);not null"`
	AdministratorToken string `gorm:"type:varchar(255);not null"`

	Messages []Message `gorm:"foreignKey:ReportID;constraint:OnDelete:CASCADE"`

	State ReportState

	Creator *string `gorm:"type:varchar(512)"` // optional email address
}

func (r *Report) IsClosed() bool {
	return r.State == ReportStateDone
}

func (r *Report) IsSpam() bool {
	return r.State == ReportStateSpam
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
	case ReportStateSpam:
		return "Spam"
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

func (r *Report) DateFmt() string {
	return r.CreatedAt.Format("02.01.2006 15:04")
}

func (m *Message) GetBody() template.HTML {
	escaped := html.EscapeString(m.Content)
	html := blackfriday.Run([]byte(escaped), blackfriday.WithExtensions(blackfriday.CommonExtensions|blackfriday.HardLineBreak))
	p := bluemonday.UGCPolicy()
	p.AllowStandardURLs()
	p.AllowAttrs("href").OnElements("a")
	p.AllowElements("b", "br", "strong", "p", "ul", "li")
	sanitized := p.Sanitize(string(html))
	return template.HTML(sanitized)
}
