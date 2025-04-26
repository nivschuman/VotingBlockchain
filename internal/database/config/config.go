package db_config

import (
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var gormConfig *gorm.Config

func GetGormConfig() *gorm.Config {
	if gormConfig == nil {
		gormConfig = &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		}
	}
	return gormConfig
}
