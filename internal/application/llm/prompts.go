package llm

import (
	"io/fs"
	"strings"
	gotemplate "text/template"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Prompts struct {
	templates *gotemplate.Template
}

func MustPrompts(fsys fs.FS, patterns ...string) *Prompts {
	return &Prompts{templates: gotemplate.Must(gotemplate.ParseFS(fsys, patterns...))}
}

func (p *Prompts) Render(name string, data any) (system, user string, err error) {
	system, err = p.One(name+".system", data)
	if err != nil {
		return "", "", err
	}
	user, err = p.One(name+".user", data)
	if err != nil {
		return "", "", err
	}
	return system, user, nil
}

func (p *Prompts) One(name string, data any) (string, error) {
	var builder strings.Builder
	if err := p.templates.ExecuteTemplate(&builder, name, data); err != nil {
		return "", errors.Wrap(err, errors.Internal, "render the prompt "+name)
	}

	rendered := strings.TrimSpace(builder.String())
	if rendered == "" {
		return "", errors.New(errors.Internal, "the prompt "+name+" rendered empty")
	}
	return rendered, nil
}
