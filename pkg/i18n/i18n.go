package i18n

import "encoding/json"

type I18n map[string]Translatable

func New(data string) (I18n, error) {
	var r I18n
	err := json.Unmarshal([]byte(data), &r)
	return r, err
}

// T returns the translation for the given key in the given language.
func (i I18n) T(lang, key string) string {
	translatable, ok := i[key]
	if !ok {
		return "err_missing_translation"
	}
	return translatable.Tr(lang)
}

type Translatable struct {
	En string `json:"en" yaml:"en"`
	De string `json:"de" yaml:"de"`
}

func (t Translatable) Tr(lang string) string {
	if lang == "en" {
		return t.En
	}
	return t.De
}
