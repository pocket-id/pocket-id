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

// GetTemplate retrieves a template instance from the provided TemplateMap using
// the Template's Path. It returns the stored *U value for that path, or nil if
// the path is not present in the map.
func GetTemplate[U any, V any](templateMap TemplateMap[U], template Template[V]) *U {
	return templateMap[template.Path]
}

// PrepareTextTemplates parses text email templates from the embedded filesystem and
// returns a map of parsed templates keyed by the provided template names.
//
// Each name in `templates` is loaded from the embedded path
// `email-templates/<name>_text.tmpl`. On success the returned map contains
// entries mapping the original name to its *text/template.Template. If any file
// fails to parse the function returns an error describing which template failed.
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

// PrepareHTMLTemplates parses HTML templates from the embedded resources filesystem.
// 
// PrepareHTMLTemplates expects a slice of template base names (without suffix). For
// each name it loads and parses "email-templates/<name>_html.tmpl" from resources.FS
// and returns a map from the base name to the parsed *html/template.Template.
//
// If any template fails to parse the function returns an error describing which
// template failed.
func PrepareHTMLTemplates(templates []string) (map[string]*htemplate.Template, error) {
	htmlTemplates := make(map[string]*htemplate.Template, len(templates))
	for _, tmpl := range templates {
		filename := tmpl + "_html.tmpl"
		templatePath := path.Join("email-templates", filename)

		parsedTemplate, err := htemplate.ParseFS(resources.FS, templatePath)
		if err != nil {
			return nil, fmt.Errorf("parsing template '%s': %w", tmpl, err)
		}

		htmlTemplates[tmpl] = parsedTemplate
	}

	return htmlTemplates, nil
}
