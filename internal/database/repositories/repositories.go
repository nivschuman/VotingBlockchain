package repositories

import "gorm.io/gorm"

func InitializeGlobalRepositories(db *gorm.DB) error {
	err := InitializeGlobalBlockRepository(db)
	if err != nil {
		return err
	}

	return InitializeGlobalTransactionRepository(db)
}
