package handler

import (
	"embed"
	"fmt"
	"html/template"
)

var (
	//go:embed template/*
	content embed.FS

	welcomeEmailTmpl *template.Template
)

func init() {
	b, err := content.ReadFile("template/welcome.html")
	if err != nil {
		panic(fmt.Errorf("read template welcome.html: %v", err))
	}

	welcomeEmailTmpl, err = template.New("welcome_email").Parse(string(b))
	if err != nil {
		panic(fmt.Errorf("parse template welcome.html: %v", err))
	}
}
