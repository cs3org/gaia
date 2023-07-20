package utils

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
)

func Go() string {
	g := os.Getenv("GAIA_GO")
	if g != "" {
		return g
	}
	return "go"
}

func FromGoEnv(key ...string) []string {
	if len(key) == 0 {
		return nil
	}

	var b bytes.Buffer
	args := []string{"env", "--json"}
	args = append(args, key...)
	cmd := exec.Command(Go(), args...)
	cmd.Stdout = &b
	err := cmd.Run()
	if err != nil {
		panic(err)
	}

	var env map[string]string
	if err := json.Unmarshal(b.Bytes(), &env); err != nil {
		panic(err)
	}
	values := make([]string, 0, len(env))
	for k, v := range env {
		values = append(values, k+"="+v)
	}
	return values
}

func KeyFromGoEnv(key string) string {
	values := FromGoEnv(key)
	if len(values) == 0 {
		return ""
	}
	return strings.SplitN(values[0], "=", 2)[1]
}
