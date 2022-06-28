package assests

import (
	"embed"
	"html/template"
	"io"

	"github.com/Masterminds/sprig/v3"
)

//go:embed templates/*
var TemplateFS embed.FS

//go:embed static/directory.css
var DirectoryCSS []byte

var Template = template.Must(
	template.New("").Funcs(sprig.FuncMap()).ParseFS(TemplateFS, "templates/*"),
)

type DirectoryEntry struct {
	Name      string
	Href      string
	IsDir     bool
	ClassName string
}

func RenderDirectory(out io.Writer, bucket, prefix string, entries []*DirectoryEntry) error {
	for _, entry := range entries {
		className := func() string {
			if entry.IsDir {
				return "folder"
			}
			return "file txt"
		}()
		if entry.ClassName == "" {
			entry.ClassName = className
		}
	}
	return Template.Lookup("directory.tmpl.html").Execute(out, map[string]interface{}{
		"bucket":  bucket,
		"prefix":  prefix,
		"entries": entries,
		"css":     template.CSS(DirectoryCSS),
	})
}
