package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	config "github.com/nivschuman/VotingBlockchain/internal/config"
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

	//Test Environment
	environment := os.Getenv("ENVIRONMENT")
	if environment == "test" {
		log.Println("|Main| Running in test environment")
		test.SetupTestEnvironmentConstants()
	}

	//Build node
	nodeBuilder, err := nodes.NewNodeBuilderImpl(conf)
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
		appBuilder := app.NewAppBuilderImpl(conf, node)
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
