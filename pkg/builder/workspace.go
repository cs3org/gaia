package builder

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cs3org/gaia/internal/utils"
	"github.com/rs/zerolog"
)

type workspace struct {
	folder string   // temp directory where all the ops are executed
	goenv  []string // environment used for go commands
	log    *zerolog.Logger
}

func (b *Builder) newWorkspace() (*workspace, error) {
	tmpFolder, err := getTempDirectory(b.TempFolder)
	if err != nil {
		return nil, fmt.Errorf("error creating temp directory: %w", err)
	}
	b.Log.Debug().Msgf("using temp folder %s as workspace", tmpFolder)
	w := &workspace{
		folder: tmpFolder,
		log:    b.Log,
	}
	return w, nil
}

func (w *workspace) setEnvKV(key, val string) {
	w.setEnv(key + "=" + val)
}

func (w *workspace) setEnv(env string) {
	es := strings.SplitN(env, "=", 2)
	if len(es) != 2 {
		panic("env variable should be of type key=val")
	}

	key := es[0]
	for i, env := range w.goenv {
		s := strings.SplitN(env, "=", 2)
		if key == s[0] {
			w.goenv[i] = env
			return
		}
	}
	// key was not found in the current env
	w.goenv = append(w.goenv, env)
}

func getTempDirectory(folder string) (string, error) {
	if folder != "" {
		err := os.MkdirAll(folder, 0755)
		return folder, err
	}
	return os.MkdirTemp("", "gaia-*")
}

func (w workspace) newCommand(ctx context.Context, cmd string, stderr io.Writer, args ...string) *exec.Cmd {
	c := exec.CommandContext(ctx, cmd, args...)
	c.Dir = w.folder
	if stderr != nil {
		c.Stderr = stderr
	}
	return c
}

func (w workspace) runGoCommand(ctx context.Context, args ...string) error {
	var buf strings.Builder
	cmd := w.newGoCommand(ctx, &buf, args...)
	w.log.Debug().Str("cmd", cmd.String()).Strs("env", cmd.Env).Send()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(buf.String()))
	}
	return nil
}

func (w workspace) runGoGetCommand(ctx context.Context, repositoryPath, version string) error {
	get := repositoryPath
	if version != "" {
		get += "@" + version
	}
	return w.runGoCommand(ctx, "get", get)
}

func (w workspace) runGoBuildCommand(ctx context.Context, src, output string, args ...string) error {
	buildArgs := []string{"build", "-o", output}
	buildArgs = append(buildArgs, args...)
	buildArgs = append(buildArgs, src)
	return w.runGoCommand(ctx, buildArgs...)
}

func (w workspace) runGoModReplaceCommand(ctx context.Context, replacement []Replace) error {
	args := make([]string, 0, len(replacement))
	for _, r := range replacement {
		w.log.Info().Msgf("replace %s", r)
		args = append(args, "-replace="+r.Format())
	}
	return w.runGoCommand(ctx, args...)
}

func (w workspace) newGoCommand(ctx context.Context, stderr io.Writer, args ...string) *exec.Cmd {
	c := w.newCommand(ctx, utils.Go(), stderr, args...)
	for _, env := range utils.FromGoEnv("GOPATH", "GOMODCACHE", "GOCACHE") {
		w.setEnv(env)
	}
	c.Env = w.goenv
	return c
}

func (w workspace) CreateFile(name string) (*os.File, error) {
	path := filepath.Join(w.folder, name)
	return os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
}

func (w workspace) Close() error {
	return os.RemoveAll(w.folder)
}
