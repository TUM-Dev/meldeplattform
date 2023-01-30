package model

import (
	"github.com/TUM-Dev/meldeplattform/pkg/i18n"
)

type Base struct {
	Lang   string
	Config Config
	Tr     i18n.I18n
}

type Index struct {
	Base
	Topics []Topic
	Topic  *int
}

type ReportPage struct {
	Base
	Report          *Report
	IsAdministrator bool
}
