package main

import (
	"errors"
	"html/template"
	"io"
)

// Template names as string constants.
const (
	MainTemplate   = "main_page"
	BoardTemplate  = "board"
	ThreadTemplate = "thread"
)

// Combine compiled template with it's file representation.
type Template struct {
	Path     string
	Compiled *template.Template
}

func (t *Template) Compile() error {
	c, err := template.ParseFiles(t.Path)
	if err != nil {
		return err
	}
	t.Compiled = c

	return nil
}

// Same as executing normal template.
func (t *Template) Execute(wr io.Writer, data any) error {
	if t.Compiled == nil {
		return errors.New("template: nil pointer executed")
	}
	return t.Compiled.Execute(wr, data)
}

// Used as templates["template_name"].Execute(in, struct)
var templates = map[string]*Template{
	MainTemplate: {
		Path: "templates/main_page.tmpl",
	},
	BoardTemplate: {
		Path: "templates/board.tmpl",
	},
	ThreadTemplate: {
		Path: "templates/thread.tmpl",
	},
}

// Compile all templates at initialization.
func compileTemplates() error {
	for k := range templates {
		err := templates[k].Compile()
		if err != nil {
			return err
		}
	}

	return nil
}
