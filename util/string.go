package util

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func StringTitle(s string) string {
	return cases.Title(language.English, cases.Compact).String(s)
}
