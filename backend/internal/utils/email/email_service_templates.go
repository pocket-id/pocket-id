package email

import (
	"fmt"
	htemplate "html/template"
	"path"
	ttemplate "text/template"

	"github.com/pocket-id/pocket-id/backend/resources"
)

type Template[V any] struct {
	Path  string
	Title func(data *TemplateData[V]) string
}

type TemplateData[V any] struct {
	AppName string
	LogoURL string
	Data    *V
}

type TemplateMap[V any] map[string]*V

func GetTemplate[U any, V any](templateMap TemplateMap[U], template Template[V]) *U {
	return templateMap[template.Path]
}

func PrepareTextTemplates(templates []string) (map[string]*ttemplate.Template, error) {
	textTemplates := make(map[string]*ttemplate.Template, len(templates))
	for _, tmpl := range templates {
		filename := tmpl + "_text.tmpl"
		templatePath := path.Join("email-templates", filename)

		parsedTemplate, err := ttemplate.ParseFS(resources.FS, templatePath)
		if err != nil {
			return nil, fmt.Errorf("parsing template '%s': %w", tmpl, err)
		}

		textTemplates[tmpl] = parsedTemplate
	}

	return textTemplates, nil
}

func PrepareHTMLTemplates(templates []string) (map[string]*htemplate.Template, error) {
	htmlTemplates := make(map[string]*htemplate.Template, len(templates))
	for _, tmpl := range templates {
		filename := fmt.Sprintf("%s_html.tmpl", tmpl)
		templatePath := path.Join("email-templates", filename)

		parsedTemplate, err := htemplate.ParseFS(resources.FS, templatePath)
		if err != nil {
			return nil, fmt.Errorf("parsing template '%s': %w", tmpl, err)
		}

		htmlTemplates[tmpl] = parsedTemplate
	}

	return htmlTemplates, nil
}
