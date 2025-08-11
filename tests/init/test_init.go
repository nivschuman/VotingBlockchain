package test_init

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/nivschuman/VotingBlockchain/internal/config"
	database "github.com/nivschuman/VotingBlockchain/internal/database/connection"
	repositories "github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	difficulty "github.com/nivschuman/VotingBlockchain/internal/difficulty"
	"gorm.io/gorm"
)

var TestConfig *config.Config
var TestDb *gorm.DB
var TestBlockRepository repositories.BlockRepository
var TestTransactionRepository repositories.TransactionRepository
var TestAddressRepository repositories.AddressRepository

func SetupTests() {
	setupTestingConstants()

	testsRoot, err := getParentDirectory("tests")
	if err != nil {
		log.Fatalf("Failed to get project root: %v", err)
	}

	err = os.Chdir(testsRoot)
	if err != nil {
		log.Fatalf("Failed to get change dir to project root: %v", err)
	}

	TestConfig, err = config.LoadConfigFromFile("config/config-test.yml")
	if err != nil {
		log.Fatalf("Failed to initialize test config: %v", err)
	}
}

func SetupTestsDatabase() {
	var err error

	TestDb, err = database.GetDatabaseConnection(":memory:")
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
	}

	TestTransactionRepository = repositories.NewTransactionRepositoryImpl(TestDb)
	TestBlockRepository = repositories.NewBlockRepositoryImpl(TestDb, TestTransactionRepository)
	TestAddressRepository = repositories.NewAddressRepositoryImpl(TestDb)

	err = TestBlockRepository.Initialize()
	if err != nil {
		log.Fatalf("Failed to initialize test block repository: %v", err)
	}

	ResetTestDatabase()
}

func ResetTestDatabase() {
	err := database.ResetDatabase(TestDb)
	if err != nil {
		log.Fatalf("Failed to reset test database: %v", err)
	}

	err = TestBlockRepository.Initialize()
	if err != nil {
		log.Fatalf("Failed to initialize test block repository: %v", err)
	}
}

func CloseTestDatabase() {
	err := database.CloseDatabaseConnection(TestDb)
	if err != nil {
		log.Fatalf("Failed to close test database: %v", err)
	}
}

func SetupTestEnvironmentConstants() {
	difficulty.MINIMUM_DIFFICULTY = uint32(0x1d80ffff)
	difficulty.TARGET_SPACING = int64(5 * 60)
	difficulty.TARGET_TIMESPAN = 6 * difficulty.TARGET_SPACING
	difficulty.INTERVAL = difficulty.TARGET_TIMESPAN / difficulty.TARGET_SPACING
	difficulty.MIN_TIMESPAN = difficulty.TARGET_TIMESPAN / 4
	difficulty.MAX_TIMESPAN = difficulty.TARGET_TIMESPAN * 4
}

func setupTestingConstants() {
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
