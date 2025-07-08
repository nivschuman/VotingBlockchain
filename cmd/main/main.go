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

	err = db.InitializeGlobalDB(dbFile)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
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
}
