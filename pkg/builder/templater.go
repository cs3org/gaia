package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

func writeMainWithPlugins(f *os.File, plugins []Plugin) error {
	plugins = slices.DeleteFunc(plugins, func(p Plugin) bool { return p.RepositoryPath == revaRepository })
	return mainTemplate.Execute(f, struct {
		Plugins  []Plugin
		RevaRepo string
	}{
		Plugins:  plugins,
		RevaRepo: revaRepository,
	})
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

var mainTemplate = template.Must(template.New("main.go").Parse(`package main

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
