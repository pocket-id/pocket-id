package email

import (
	"fmt"
	htemplate "html/template"
	"io/fs"
	"path"
	ttemplate "text/template"
)

const templateComponentsDir = "components"

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

type clonable[V pareseable[V]] interface {
	Clone() (V, error)
}

type pareseable[V any] interface {
	ParseFS(fs.FS, ...string) (V, error)
}

func prepareTemplate[V pareseable[V]](template string, rootTemplate clonable[V], templateDir fs.FS, suffix string) (V, error) {
	tmpl, err := rootTemplate.Clone()
	if err != nil {
		return *new(V), fmt.Errorf("clone root html template: %w", err)
	}

	filename := fmt.Sprintf("%s%s", template, suffix)
	_, err = tmpl.ParseFS(templateDir, filename)
	if err != nil {
		return *new(V), fmt.Errorf("parsing html template '%s': %w", template, err)
	}

	return tmpl, nil
}

func PrepareTextTemplates(templateDir fs.FS, templates []string) (map[string]*ttemplate.Template, error) {
	components := path.Join(templateComponentsDir, "*_text.tmpl")
	rootTmpl, err := ttemplate.ParseFS(templateDir, components)
	if err != nil {
		return nil, fmt.Errorf("unable to parse templates '%s': %w", components, err)
	}

	var textTemplates = make(map[string]*ttemplate.Template, len(templates))
	for _, tmpl := range templates {
		rootTmplClone, err := rootTmpl.Clone()
		if err != nil {
			return nil, fmt.Errorf("clone root template: %w", err)
		}

		textTemplates[tmpl], err = prepareTemplate[*ttemplate.Template](tmpl, rootTmplClone, templateDir, "_text.tmpl")
		if err != nil {
			return nil, fmt.Errorf("parse '%s': %w", tmpl, err)
		}
	}

	return textTemplates, nil
}

func PrepareHTMLTemplates(templateDir fs.FS, templates []string) (map[string]*htemplate.Template, error) {
	components := path.Join(templateComponentsDir, "*_html.tmpl")
	rootTmpl, err := htemplate.ParseFS(templateDir, components)
	if err != nil {
		return nil, fmt.Errorf("unable to parse templates '%s': %w", components, err)
	}

	var htmlTemplates = make(map[string]*htemplate.Template, len(templates))
	for _, tmpl := range templates {
		rootTmplClone, err := rootTmpl.Clone()
		if err != nil {
			return nil, fmt.Errorf("clone root template: %w", err)
		}

		htmlTemplates[tmpl], err = prepareTemplate[*htemplate.Template](tmpl, rootTmplClone, templateDir, "_html.tmpl")
		if err != nil {
			return nil, fmt.Errorf("parse '%s': %w", tmpl, err)
		}
	}

	return htmlTemplates, nil
}
