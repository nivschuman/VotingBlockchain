package db_connection

import (
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

func InitializeGlobalDB(dbFile string) error {
	if GlobalDB != nil {
		return nil
	}

	var err error
	GlobalDB, err = GetDatabaseConnection(dbFile)

	return err
}

func GetDatabaseConnection(dbFile string) (*gorm.DB, error) {
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
