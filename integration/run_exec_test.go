package integration

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/magefile/mage/sh"
)

func TestIntegration(t *testing.T) {
	projectName := strings.ToLower(t.Name())
	outputDir := t.TempDir()
	composeFile := "./test-run-exec/docker-compose.yml"
	localExecFilename := "local_exec.txt"
	execFilename := "exec.txt"
	runFilename := "run.txt"

	sleepForSec := 10
	scheduleEverySec := 3
	expectedExecutions := (sleepForSec / scheduleEverySec)

	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		t.Fatalf("Compose file %s not found", composeFile)
	}

	t.Setenv("COMPOSE_FILE", composeFile)
	t.Setenv("COMPOSE_PROJECT_NAME", projectName)
	t.Setenv("OUTPUT_DIR", outputDir)
	t.Setenv("SCHEDULE", fmt.Sprintf("@every %ds", scheduleEverySec))
	t.Setenv("SLEEP_FOR", strconv.Itoa(sleepForSec))
	t.Setenv("LOCAL_EXEC_OUTPUT_FILE", localExecFilename)
	t.Setenv("EXEC_OUTPUT_FILE", execFilename)
	t.Setenv("RUN_OUTPUT_FILE", runFilename)

	for _, command := range []string{"config", "build", "pull"} {
		t.Run("docker compose "+command, func(t *testing.T) {
			t.Logf("Running docker compose %s", command)
			if err := sh.RunV("docker", "compose", command); err != nil {
				t.Fatal(err)
			}
		})
	}

	t.Run("docker compose up", func(t *testing.T) {
		if err := sh.RunV("docker", "compose", "up", "--exit-code-from", "sleep1"); err != nil {
			t.Fatal(err)
		}

		for _, file := range []string{localExecFilename, execFilename, runFilename} {
			t.Log("Checking for outputs in", file)
			count, content, err := checkFile(filepath.Join(outputDir, file))
			if err != nil {
				t.Error(err)
				continue
			}

			if count != expectedExecutions {
				t.Errorf("expected %d lines in %s, but got %d. File content:\n%s", expectedExecutions, file, count, content)
			}
		}
	})
}

func checkFile(path string) (int, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	content := strings.Builder{}
	count := 0
	for scanner.Scan() {
		content.Write(scanner.Bytes())
		count++
	}

	if err := scanner.Err(); err != nil {
		return 0, "", err
	}

	return count, content.String(), nil
}
