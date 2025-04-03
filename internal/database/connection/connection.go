package db_connection

import (
	"errors"
	"fmt"
	"log"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

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
	GlobalDB, err = GetAppDatabaseConnection()

	return err
}

func GetAppDatabaseConnection() (*gorm.DB, error) {
	env := os.Getenv("APP_ENV")

	if env == "" {
		return nil, errors.New("APP_ENV environment variable not set")
	}

	dbFile := fmt.Sprintf("databases/blockchain-%s.db", env)
	db, err := gorm.Open(sqlite.Open(dbFile), &gorm.Config{})

	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(modelsToMigrate...)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func ResetDatabase(db *gorm.DB) error {
	err := db.Migrator().DropTable(modelsToMigrate...)

	if err != nil {
		return err
	}

	return db.AutoMigrate(modelsToMigrate...)
}

func DeleteDatabase() error {
	env := os.Getenv("APP_ENV")

	if env == "" {
		return errors.New("APP_ENV environment variable not set")
	}

	dbFile := fmt.Sprintf("databases/blockchain-%s.db", env)
	err := os.Remove(dbFile)

	if err != nil {
		return fmt.Errorf("failed to delete database file: %w", err)
	}

	log.Printf("Database file '%s' deleted successfully", dbFile)
	return nil
}
