package db_connection

import (
	"errors"
	"fmt"
	"log"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	config "github.com/nivschuman/VotingBlockchain/internal/database/config"
	models "github.com/nivschuman/VotingBlockchain/internal/database/models"
)

var modelsToMigrate = []any{
	&models.TransactionDB{},
	&models.BlockDB{},
	&models.BlockHeaderDB{},
	&models.TransactionBlockDB{},
}

var GlobalDB *gorm.DB = nil

func InitializeGlobalDB() error {
	if GlobalDB != nil {
		return nil
	}

	var err error
	GlobalDB, err = GetDatabaseConnection()

	return err
}

func GetDatabaseConnection() (*gorm.DB, error) {
	env := os.Getenv("APP_ENV")

	if env == "" {
		return nil, errors.New("APP_ENV environment variable not set")
	}

	dbFile := "databases/blockchain.db"

	if env != "test" {
		dir := "databases"
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, os.ModePerm); err != nil {
				return nil, fmt.Errorf("failed to create databases directory: %w", err)
			}
			log.Printf("Created directory '%s'", dir)
		}
	} else if env == "test" {
		dbFile = ":memory:"
		log.Println("Using in-memory database for testing")
	}

	db, err := gorm.Open(sqlite.Open(dbFile), config.GetGormConfig())

	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(modelsToMigrate...)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func CloseDatabaseConnection(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

func ResetDatabase(db *gorm.DB) error {
	err := db.Migrator().DropTable(modelsToMigrate...)

	if err != nil {
		return err
	}

	return db.AutoMigrate(modelsToMigrate...)
}
