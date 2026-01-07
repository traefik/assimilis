package generator

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	texttpl "text/template"
)

//go:embed templates/*.gotpl
var embeddedTemplates embed.FS

func renderHTML(cfg Config, model any) (string, error) {
	var tpl *template.Template
	var err error

	if cfg.HTMLTemplatePath != "" {
		tpl, err = template.ParseFiles(cfg.HTMLTemplatePath)
	} else {
		tpl, err = template.ParseFS(embeddedTemplates, "templates/third_party_licenses.gotpl")
	}

	if err != nil {
		return "", fmt.Errorf("failed to parse HTML template: %w", err)
	}

	var buf bytes.Buffer

	if err := tpl.Execute(&buf, model); err != nil {
		return "", fmt.Errorf("failed to execute HTML template: %w", err)
	}

	return buf.String(), nil
}

func renderText(cfg Config, model any) (string, error) {
	var tpl *texttpl.Template
	var err error

	if cfg.NoticeTplPath != "" {
		tpl, err = texttpl.ParseFiles(cfg.NoticeTplPath)
	} else {
		tpl, err = texttpl.ParseFS(embeddedTemplates, "templates/notice.gotpl")
	}

	if err != nil {
		return "", fmt.Errorf("failed to parse notice template: %w", err)
	}

	var buf bytes.Buffer

	if err := tpl.Execute(&buf, model); err != nil {
		return "", fmt.Errorf("failed to execute notice template: %w", err)
	}

	return buf.String(), nil
}
