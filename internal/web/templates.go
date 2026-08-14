package web

import (
	"html/template"
	"io/fs"
	"path"
	"strings"
)

// templateSet 是全部模板的集合（layout + 页面 + partials）。
var templateSet *template.Template

func init() {
	templateSet = template.New("root").Funcs(template.FuncMap{})
	fs.WalkDir(assets, "templates", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".html") {
			return nil
		}
		content, err := assets.ReadFile(p)
		if err != nil {
			return err
		}
		_, err = templateSet.New(path.Base(p)).Parse(string(content))
		return err
	})
}

func templateNew(name string) (*template.Template, error) {
	t := templateSet.Lookup(name + ".html")
	if t == nil {
		t = templateSet.Lookup(name)
	}
	if t == nil {
		return nil, fs.ErrNotExist
	}
	return t, nil
}
