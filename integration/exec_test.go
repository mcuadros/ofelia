package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/magefile/mage/sh"
)

func TestExec(t *testing.T) {
	projectName := strings.ToLower(t.Name())
	outputDir := t.TempDir()
	local_exec_filename := "local_exec.txt"
	exec_filename := "exec.txt"
	sleep_for_seconds := 15
	schedule_seconds := 3

	t.Setenv("COMPOSE_FILE", "./test-exec/docker-compose.yml")
	t.Setenv("COMPOSE_PROJECT_NAME", projectName)
	t.Setenv("OUTPUT_DIR", outputDir)
	t.Setenv("SCHEDULE", fmt.Sprintf("@every %ds", schedule_seconds))
	t.Setenv("SLEEP_SEC", strconv.Itoa(sleep_for_seconds))
	t.Setenv("LOCAL_EXEC_OUTPUT_FILE", local_exec_filename)
	t.Setenv("EXEC_OUTPUT_FILE", exec_filename)

	t.Log("Printing docker compose config")
	if err := sh.RunV("docker", "compose", "config"); err != nil {
		t.Fatal(err)
	}

	t.Log("Running docker compose up")
	if err := sh.RunV("docker", "compose", "up", "--exit-code-from", "sleep1"); err != nil {
		t.Error(err)
	}

	for _, file := range []string{local_exec_filename, exec_filename} {
		t.Log("Checking for outputs in", file)
		data, err := os.ReadFile(filepath.Join(outputDir, file))
		if err != nil {
			t.Error(err)
		}

		split := strings.Split(string(data), "\n")
		if expectedLines := sleep_for_seconds / schedule_seconds; len(split) != expectedLines {
			t.Errorf("expected %d lines in %s, but got %d", expectedLines, file, len(split))
		}
	}
}
