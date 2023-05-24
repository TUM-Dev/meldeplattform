package model

import (
	"github.com/TUM-Dev/meldeplattform/pkg/i18n"
	"html/template"
)

type Base struct {
	Lang   string
	Config Config
	Tr     i18n.I18n
	Topics []Topic

	LoggedIn bool
	Name     string
	Email    string
	UID      string
	IsAdmin  bool
}

type InfoPage struct {
	Base
	Content template.HTML
}

type Index struct {
	Base
	Topic *Topic
	Token string
}

type NewTopicPage struct {
	Base
}

type ReportPage struct {
	Base
	Report          *Report
	IsAdministrator bool
}

type ReportsOfTopicPage struct {
	Base
	Topic   *Topic
	Reports []Report
}
