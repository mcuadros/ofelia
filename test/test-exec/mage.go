//go:build mage

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/sh"
)

// Runs go mod download and then installs the binary.
func Test() error {
	fmt.Println("Printing docker compose config")
	if err := sh.RunV("docker", "compose", "config"); err != nil {
		return err
	}

	fmt.Println("Running docker compose up")
	if err := sh.RunV("docker", "compose", "up", "--exit-code-from", "sleep1"); err != nil {
		return err
	}

	fmt.Println("Checking for outputs")
	od := os.Getenv("OUTPUT_DIR")
	projectName := os.Getenv("COMPOSE_PROJECT_NAME")
	data, err := os.ReadFile(filepath.Join(od, projectName, "sleep1.txt"))
	if err != nil {
		return err
	}

	split := strings.Split(string(data), "\n")
	if len(split) < 2 {
		return errors.New("expected at least 2 lines in sleep1.txt")
	}

	return nil
}
