package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
	database "github.com/nivschuman/VotingBlockchain/internal/database/connection"
	nodes "github.com/nivschuman/VotingBlockchain/internal/nodes"
	app "github.com/nivschuman/VotingBlockchain/internal/ui/app"
	test "github.com/nivschuman/VotingBlockchain/tests/init"
)

func main() {
	//Load configuration
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		configFile = "config/config.yml"
	}

	conf, err := config.LoadConfigFromFile(configFile)
	if err != nil {
		log.Fatalf("Failed to load config file: %v", err)
	}

	//Load database
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

	db, err := database.GetDatabaseConnection(dbFile)
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
	}

	//Reset database in test environment
	if environment == "test" {
		err = database.ResetDatabase(db)
		if err != nil {
			log.Fatalf("Failed to reset test database: %v", err)
		}
	}

	//Database cleanup
	defer func() {
		err := database.CloseDatabaseConnection(db)
		if err != nil {
			log.Fatalf("Failed to close database connection: %v", err)
		}
	}()

	//Build node
	nodeBuilder, err := nodes.NewNodeBuilderImpl(db, conf)
	if err != nil {
		log.Fatalf("Failed to create node builder: %v", err)
	}

	node, err := nodeBuilder.BuildNode()
	if err != nil {
		log.Fatalf("Failed to build node: %v", err)
	}

	//Start node
	go node.Start()
	if conf.UiConfig.Enabled {
		appBuilder := app.NewAppBuilderImpl(db)
		mainApp := appBuilder.BuildApp()
		mainApp.Start()
	} else {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		<-ctx.Done()
	}

	log.Println("|Main| Shutting down node...")
	node.Stop()
}
