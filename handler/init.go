package handler

import (
	"embed"
	"fmt"
	"html/template"
)

var (
	//go:embed template/*
	content embed.FS

	initUserTmpl *template.Template
)

func init() {
	b, err := content.ReadFile("template/init_user.html")
	if err != nil {
		panic(fmt.Errorf("read template init_user.html: %v", err))
	}

	initUserTmpl, err = template.New("init_user").Parse(string(b))
	if err != nil {
		panic(fmt.Errorf("parse template init_user.html: %v", err))
	}
}
