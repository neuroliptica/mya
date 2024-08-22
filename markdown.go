package main

import (
	"html/template"
	"strings"
)

type Tag struct {
	open    string
	closing string // optional

	replaceOpen    string
	replaceClosing string // optional
}

var markdown = []Tag{
	// bold
	{
		open:           "[b]",
		replaceOpen:    "<b>",
		closing:        "[/b]",
		replaceClosing: "</b>",
	},

	// italic
	{
		open:           "[i]",
		replaceOpen:    "<i>",
		closing:        "[/i]",
		replaceClosing: "</i>",
	},
}

// determine if pr is prefix of dest.
func prefix(dest []rune, pr []rune) bool {
	if len(dest) < len(pr) {
		return false
	}
	for i := 0; i < len(pr); i++ {
		if dest[i] != pr[i] {
			return false
		}
	}
	return true
}

// build tags right sequence.
func sequence(text string, tag *Tag) string {
	c := 0
	v := new(strings.Builder)
	f := []rune(text)
	op, cl := []rune(tag.open), []rune(tag.closing)
	for i := 0; i < len(f); i++ {
		if prefix(f[i:], op) {
			c++
		} else if prefix(f[i:], cl) {
			c--
		}
		if c < 0 {
			v.WriteString(tag.open)
			c++
		}
		v.WriteRune(f[i])
	}
	for ; c > 0; c-- {
		v.WriteString(tag.closing)
	}
	return v.String()
}

func replaceMarkdown(text string) template.HTML {
	text = template.HTMLEscapeString(text)
	for i := range markdown {
		text = sequence(text, &markdown[i])
	}
	r := []string{}
	for _, c := range markdown {
		r = append(r, c.open, c.replaceOpen)
		if c.closing != "" {
			r = append(r, c.closing, c.replaceClosing)
		}
	}
	// todo(zvezdochka): maybe should init gloabal replacer?
	e := strings.NewReplacer(r...).Replace(text)

	return template.HTML(e)
}
