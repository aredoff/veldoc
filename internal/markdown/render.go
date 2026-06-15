package markdown

import (
	"bytes"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

var (
	engine = goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
		),
	)
	sanitizer = bluemonday.UGCPolicy()
)

func Render(source string) (string, error) {
	var buf bytes.Buffer
	if err := engine.Convert([]byte(source), &buf); err != nil {
		return "", err
	}
	return string(sanitizer.SanitizeBytes(buf.Bytes())), nil
}
