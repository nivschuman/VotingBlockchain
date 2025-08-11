package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
	db "github.com/nivschuman/VotingBlockchain/internal/database/connection"
	repositories "github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	nodes "github.com/nivschuman/VotingBlockchain/internal/nodes"
	app "github.com/nivschuman/VotingBlockchain/internal/ui/app"
	test "github.com/nivschuman/VotingBlockchain/tests/init"
)

func main() {
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
		log.Println("|Main| Running in test environment")
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

	go node.Start()
	if config.GlobalConfig.UiConfig.Enabled {
		mainApp := app.MainApp()
		mainApp.Start()
	} else {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		<-ctx.Done()
	}

	log.Println("|Main| Shutting down node...")
	node.Stop()
	err = db.CloseDatabaseConnection(db.GlobalDB)
	if err != nil {
		log.Fatalf("Failed to close database connection: %v", err)
	}
}
