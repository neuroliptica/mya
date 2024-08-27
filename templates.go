package main

import (
	"html/template"
)

var templates *template.Template

// Compile all templates at initialization.
func compileTemplates() error {
	var err error
	templates, err = template.ParseGlob(
		"templates/*.tmpl",
	)
	return err
}
