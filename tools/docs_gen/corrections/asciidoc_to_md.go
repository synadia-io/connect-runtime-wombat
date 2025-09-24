package corrections

import (
	"fmt"
	"strings"

	"cuelang.org/go/pkg/regexp"
	"github.com/Jeffail/gabs/v2"
)

type AsciidocToMd struct {
	Path string `yaml:"path"`
}

func (a AsciidocToMd) Correct(doc *gabs.Container) (*gabs.Container, error) {
	v := doc.Path(a.Path)
	if v == nil {
		return doc, nil
	}

	str, ok := v.Data().(string)
	if !ok {
		return nil, fmt.Errorf("type assertion failed on container data")
	}

	_, err := doc.SetP(convertAsciidocToMarkdown(str), a.Path)
	return doc, err
}

func convertAsciidocToMarkdown(str string) string {
	lines := strings.Split(str, "\n")

	result := ""
	for _, line := range lines {
		line, _ = regexp.ReplaceAll("^== ", line, "# ")
		line, _ = regexp.ReplaceAll("^=== ", line, "## ")
		line, _ = regexp.ReplaceAll("^==== ", line, "### ")
		line, _ = regexp.ReplaceAll("^===== ", line, "#### ")
		line, _ = regexp.ReplaceAll("^====== ", line, "##### ")
		line, _ = regexp.ReplaceAll("^======= ", line, "###### ")
		result += line + "\n"
	}

	return result
}
