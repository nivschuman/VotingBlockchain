package test_init

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
	db "github.com/nivschuman/VotingBlockchain/internal/database/connection"
	repositories "github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	difficulty "github.com/nivschuman/VotingBlockchain/internal/difficulty"
)

func SetupTests() {
	setupConstants()

	testsRoot, err := getParentDirectory("tests")
	if err != nil {
		log.Fatalf("Failed to get project root: %v", err)
	}

	err = os.Chdir(testsRoot)
	if err != nil {
		log.Fatalf("Failed to get change dir to project root: %v", err)
	}

	err = config.InitializeGlobalConfig("config/config-test.yml")
	if err != nil {
		log.Fatalf("Failed to initialize global config: %v", err)
	}
}

func SetupTestsDatabase() {
	err := db.InitializeGlobalDB(":memory:")
	if err != nil {
		log.Fatalf("Failed to initialize db: %v", err)
	}

	err = repositories.InitializeGlobalRepositories(db.GlobalDB)
	if err != nil {
		log.Fatalf("Failed to initialize repositories: %v", err)
	}

	ResetTestDatabase()
}

func ResetTestDatabase() {
	err := db.ResetDatabase(db.GlobalDB)
	if err != nil {
		log.Fatalf("Failed to reset db: %v", err)
	}

	err = repositories.GlobalBlockRepository.Setup()
	if err != nil {
		log.Fatalf("Failed to setup block repository: %v", err)
	}
}

func CloseTestDatabase() {
	err := db.CloseDatabaseConnection(db.GlobalDB)
	if err != nil {
		log.Fatalf("Failed to close test database: %v", err)
	}
}

func setupConstants() {
	difficulty.MINIMUM_DIFFICULTY = uint32(0x207fffff)
	difficulty.TARGET_TIMESPAN = int64(10 * 60)
	difficulty.TARGET_SPACING = int64(1 * 60)
	difficulty.INTERVAL = difficulty.TARGET_TIMESPAN / difficulty.TARGET_SPACING
	difficulty.MIN_TIMESPAN = difficulty.TARGET_TIMESPAN / 4
	difficulty.MAX_TIMESPAN = difficulty.TARGET_TIMESPAN * 4
}

func getParentDirectory(directoryName string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if filepath.Base(dir) == directoryName {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("directory '%s' not found in path", directoryName)
}
