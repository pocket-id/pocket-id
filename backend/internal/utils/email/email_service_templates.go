package email

import (
	"fmt"
	htemplate "html/template"
	"io/fs"
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

type cloneable[V pareseable[V]] interface {
	Clone() (V, error)
}

type pareseable[V any] interface {
	ParseFS(fs.FS, ...string) (V, error)
}

func prepareTemplate[V pareseable[V]](templateFS fs.FS, template string, rootTemplate cloneable[V], suffix string) (V, error) {
	tmpl, err := rootTemplate.Clone()
	if err != nil {
		return *new(V), fmt.Errorf("clone root template: %w", err)
	}

	filename := fmt.Sprintf("%s%s", template, suffix)
	templatePath := path.Join("email-templates", filename)
	_, err = tmpl.ParseFS(templateFS, templatePath)
	if err != nil {
		return *new(V), fmt.Errorf("parsing template '%s': %w", template, err)
	}

	return tmpl, nil
}

func PrepareTextTemplates(templates []string) (map[string]*ttemplate.Template, error) {
	components := path.Join("email-templates", "components", "*_text.tmpl")
	rootTmpl, err := ttemplate.ParseFS(resources.FS, components)
	if err != nil {
		return nil, fmt.Errorf("unable to parse templates '%s': %w", components, err)
	}

	textTemplates := make(map[string]*ttemplate.Template, len(templates))
	for _, tmpl := range templates {
		rootTmplClone, err := rootTmpl.Clone()
		if err != nil {
			return nil, fmt.Errorf("clone root template: %w", err)
		}

		textTemplates[tmpl], err = prepareTemplate[*ttemplate.Template](resources.FS, tmpl, rootTmplClone, "_text.tmpl")
		if err != nil {
			return nil, fmt.Errorf("parse '%s': %w", tmpl, err)
		}
	}

	return textTemplates, nil
}

func PrepareHTMLTemplates(templates []string) (map[string]*htemplate.Template, error) {
	components := path.Join("email-templates", "components", "*_html.tmpl")
	rootTmpl, err := htemplate.ParseFS(resources.FS, components)
	if err != nil {
		return nil, fmt.Errorf("unable to parse templates '%s': %w", components, err)
	}

	htmlTemplates := make(map[string]*htemplate.Template, len(templates))
	for _, tmpl := range templates {
		rootTmplClone, err := rootTmpl.Clone()
		if err != nil {
			return nil, fmt.Errorf("clone root template: %w", err)
		}

		htmlTemplates[tmpl], err = prepareTemplate[*htemplate.Template](resources.FS, tmpl, rootTmplClone, "_html.tmpl")
		if err != nil {
			return nil, fmt.Errorf("parse '%s': %w", tmpl, err)
		}
	}

	return htmlTemplates, nil
}
