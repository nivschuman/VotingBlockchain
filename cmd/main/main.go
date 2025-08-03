package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
	db "github.com/nivschuman/VotingBlockchain/internal/database/connection"
	repositories "github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	nodes "github.com/nivschuman/VotingBlockchain/internal/nodes"
	test "github.com/nivschuman/VotingBlockchain/tests/init"
)

func main() {
	log.Printf("Start number of goroutines: %d", runtime.NumGoroutine())

	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		configFile = "config/config.yml"
	}

	err := config.InitializeGlobalConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load config file: %v", err)
	}

	dbFile := os.Getenv("DATABASE_FILE")
	if dbFile == "" {
		dbFile = "databases/blockchain.db"
	}

	environment := os.Getenv("ENVIRONMENT")
	if environment == "test" {
		log.Println("Running in test environment")
		dbFile = "databases/blockchain-test.db"
		test.SetupTestEnvironmentConstants()
	}

	err = db.InitializeGlobalDB(dbFile)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	if environment == "test" {
		err = db.ResetDatabase(db.GlobalDB)
		if err != nil {
			log.Fatalf("Failed to reset test database: %v", err)
		}
	}

	err = repositories.InitializeGlobalRepositories(db.GlobalDB)
	if err != nil {
		log.Fatalf("Failed to initialize repositories: %v", err)
	}

	err = repositories.GlobalBlockRepository.Setup()
	if err != nil {
		log.Fatalf("Failed to setup block repository: %v", err)
	}

	node, err := nodes.GlobalNodeFactory.CreateNode(nodes.NodeType(config.GlobalConfig.NodeConfig.Type))
	if err != nil {
		log.Fatalf("Failed to create node: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go node.Start()

	<-ctx.Done()
	node.Stop()

	err = db.CloseDatabaseConnection(db.GlobalDB)
	if err != nil {
		log.Fatalf("Failed to close database connection: %v", err)
	}

	fmt.Println("End number of goroutines:", runtime.NumGoroutine())
	buf := make([]byte, 1<<20)
	stacklen := runtime.Stack(buf, true)
	log.Printf("Goroutine dump:\n%s", buf[:stacklen])
}
