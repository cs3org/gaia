package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"os"
	"os/exec"
	"slices"
	"text/template"

	"github.com/cs3org/gaia/internal/utils"
)

type Plugin struct {
	RepositoryPath string
	Version        string
}

func (p Plugin) String() string {
	s := p.RepositoryPath
	if p.Version != "" {
		s += "@" + p.Version
	}
	return s
}

type Replace struct {
	From      string
	To        string
	ToVersion string
}

// Format formats the replacement string to be valid
// with the go mod edit command.
func (r Replace) Format() string {
	s := r.From + "=" + r.To
	if r.ToVersion != "" {
		s += "@" + r.ToVersion
	}
	return s
}

func (r Replace) String() string {
	s := r.From + " => " + r.To
	if r.ToVersion != "" {
		s += "@" + r.ToVersion
	}
	return s
}

// Binary is the program that gaia builds.
type Binary string

const (
	// Revad is the reva daemon, github.com/cs3org/reva/v3/cmd/revad. It is
	// the default.
	Revad Binary = "revad"
	// Reva is the reva CLI, github.com/cs3org/reva/v3/cmd/reva/app.
	Reva Binary = "reva"
)

// ParseBinary returns the binary named s: "revad" or "reva". An empty s is
// revad.
func ParseBinary(s string) (Binary, error) {
	switch b := Binary(s); b {
	case "":
		return Revad, nil
	case Revad, Reva:
		return b, nil
	}
	return "", fmt.Errorf("unknown binary %q: it is revad or reva", s)
}

const (
	// brandFile is the brand in the workspace, which main.go embeds.
	brandFile = "brand.toml"
	// brandCheckArg makes the reva CLI check its brand and its plugins, and
	// exit 0 or 1.
	brandCheckArg = "__brandcheck"
)

// writeMain writes the main.go of the binary, with a blank import for each
// plugin. With brand, the reva CLI embeds brandFile and runs with that brand.
func writeMain(w io.Writer, binary Binary, plugins []Plugin, brand bool) error {
	if brand && binary != Reva {
		return fmt.Errorf("a brand is for the reva binary, not %s", binary)
	}
	// DeleteFunc clears the end of the slice it changes: work on a copy,
	// the caller still needs every plugin.
	plugins = slices.DeleteFunc(slices.Clone(plugins), func(p Plugin) bool { return p.RepositoryPath == revaRepository })
	t := revadTemplate
	if binary == Reva {
		t = revaTemplate
	}
	data := struct {
		Plugins   []Plugin
		RevaRepo  string
		BrandFile string
	}{
		Plugins:  plugins,
		RevaRepo: revaRepository,
	}
	if brand {
		data.BrandFile = brandFile
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return err
	}
	src, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("error formatting main.go: %w", err)
	}
	_, err = w.Write(src)
	return err
}

type GoMod struct {
	Module struct {
		Path string `json:"Path"`
	} `json:"Module"`
	Go      string `json:"Go"`
	Require []struct {
		Path     string `json:"Path"`
		Version  string `json:"Version"`
		Indirect bool   `json:"Indirect,omitempty"`
	} `json:"Require"`
	Exclude any `json:"Exclude"`
	Replace []struct {
		Old struct {
			Path string `json:"Path"`
		} `json:"Old"`
		New struct {
			Path    string `json:"Path"`
			Version string `json:"Version"`
		} `json:"New"`
	} `json:"Replace"`
	Retract any `json:"Retract"`
}

func parseGoModFile(ctx context.Context, path string) (*GoMod, error) {
	var stdout bytes.Buffer

	c := exec.CommandContext(ctx, utils.Go(), "mod", "edit", "-json", path)
	c.Stdout = &stdout

	if err := c.Run(); err != nil {
		return nil, fmt.Errorf("error running command: %w", err)
	}

	var gomod GoMod
	if err := json.NewDecoder(&stdout).Decode(&gomod); err != nil {
		return nil, fmt.Errorf("error decoding go.mod: %w", err)
	}
	return &gomod, nil
}

func isRevaLocalReplacement(repl []Replace) (string, bool) {
	for _, r := range repl {
		if r.From == revaRepository {
			_, err := os.Stat(r.To)
			return r.To, err == nil
		}
	}
	return "", false
}

var revadTemplate = template.Must(template.New("main.go").Parse(`package main

import (
	revadcmd "{{.RevaRepo}}/cmd/revad"
{{- range .Plugins }}
	_ "{{ .RepositoryPath }}"
{{- end }}
)

func main() {
	revadcmd.Main()
}
`))

var revaTemplate = template.Must(template.New("main.go").Parse(`package main

import (
{{- if .BrandFile }}
	_ "embed"

	"{{.RevaRepo}}/pkg/cli/brand"
{{- end }}
	revacmd "{{.RevaRepo}}/cmd/reva/app"
{{- range .Plugins }}
	_ "{{ .RepositoryPath }}"
{{- end }}
)
{{ if .BrandFile }}
// embedded is the brand of the program (gaia build --brand).
//
//go:embed {{ .BrandFile }}
var embedded []byte
{{ end }}
func main() {
{{- if .BrandFile }}
	revacmd.Main(revacmd.WithBrand(brand.MustParse(embedded)))
{{- else }}
	revacmd.Main()
{{- end }}
}
`))
