package test_init

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
	db "github.com/nivschuman/VotingBlockchain/internal/database/connection"
)

func InitializeTestDatabase() {
	err := db.InitializeGlobalDB()

	if err != nil {
		log.Fatalf("Failed to initialize db: %v", err)
	}

	err = db.ResetDatabase(db.GlobalDB)

	if err != nil {
		log.Fatalf("Failed to initialize db: %v", err)
	}
}

func init() {
	err := os.Setenv("APP_ENV", "test")

	if err != nil {
		log.Fatalf("Failed to set APP_ENV: %v", err)
	}

	projectRoot, err := getProjectRoot()

	if err != nil {
		log.Fatalf("Failed to get project root: %v", err)
	}

	os.Chdir(projectRoot)

	err = config.InitializeGlobalConfig()

	if err != nil {
		log.Fatalf("Failed to initialize global config: %v", err)
	}
}

func getProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		dir = filepath.Dir(dir)

		if dir == "/" || dir == "." {
			return "", fmt.Errorf("could not find project root (go.mod not found)")
		}
	}
}
