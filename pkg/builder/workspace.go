package builder

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cs3org/gaia/internal/utils"
)

type workspace struct {
	folder string   // temp directory where all the ops are executed
	goenv  []string // environment used for go commands
}

func (b *Builder) newWorkspace() (*workspace, error) {
	tmpFolder, err := getTempDirectory(b.TempFolder)
	if err != nil {
		return nil, fmt.Errorf("error creating temp directory: %w", err)
	}
	log.Println("[INFO] Using temp folder", tmpFolder, "as workspace")
	w := &workspace{
		folder: tmpFolder,
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

func (w workspace) newCommand(ctx context.Context, cmd string, args ...string) *exec.Cmd {
	c := exec.CommandContext(ctx, cmd, args...)
	c.Dir = w.folder
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c
}

func (w workspace) runGoCommand(ctx context.Context, args ...string) error {
	cmd := w.newGoCommand(ctx, args...)
	return cmd.Run()
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
		args = append(args, "-replace="+r.Format())
	}
	return w.runGoCommand(ctx, args...)
}

func (w workspace) newGoCommand(ctx context.Context, args ...string) *exec.Cmd {
	c := w.newCommand(ctx, utils.Go(), args...)
	for _, env := range utils.FromGoEnv("GOPATH", "GOMODCACHE", "GOCACHE") {
		w.setEnv(env)
	}
	c.Env = w.goenv
	log.Printf("[INFO] cmd=%s env=%v\n", c.String(), c.Env)
	return c
}

func (w workspace) CreateFile(name string) (*os.File, error) {
	path := filepath.Join(w.folder, name)
	return os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
}

func (w workspace) Close() error {
	return os.RemoveAll(w.folder)
}
