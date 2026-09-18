package content

import (
	"embed"
	"strings"
	gotemplate "text/template"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

//go:embed prompts/*.tmpl
var promptFS embed.FS

var prompts = gotemplate.Must(gotemplate.ParseFS(promptFS, "prompts/*.tmpl"))

func render(name string, data any) (system, user string, err error) {
	system, err = renderOne(name+".system", data)
	if err != nil {
		return "", "", err
	}
	user, err = renderOne(name+".user", data)
	if err != nil {
		return "", "", err
	}
	return system, user, nil
}

func renderOne(name string, data any) (string, error) {
	var builder strings.Builder
	if err := prompts.ExecuteTemplate(&builder, name, data); err != nil {
		return "", errors.Wrap(err, errors.Internal, "render the prompt "+name)
	}

	rendered := strings.TrimSpace(builder.String())
	if rendered == "" {
		return "", errors.New(errors.Internal, "the prompt "+name+" rendered empty")
	}
	return rendered, nil
}
