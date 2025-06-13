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

func SetupTestsDatabase() {
	err := db.InitializeGlobalDB()

	if err != nil {
		log.Fatalf("Failed to initialize db: %v", err)
	}

	err = initializeRepositories()

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

	genesisBlock := repositories.GlobalBlockRepository.GenesisBlock()
	err = repositories.GlobalBlockRepository.InsertIfNotExists(genesisBlock)

	if err != nil {
		log.Fatalf("Failed to insert genesis block: %v", err)
	}

	err = repositories.GlobalBlockRepository.SetActiveChainTipId()

	if err != nil {
		log.Fatalf("Failed to set active chain tip: %v", err)
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

func initializeRepositories() error {
	err := repositories.InitializeGlobalBlockRepository(db.GlobalDB)

	if err != nil {
		return err
	}

	return repositories.InitializeGlobalTransactionRepository(db.GlobalDB)
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
