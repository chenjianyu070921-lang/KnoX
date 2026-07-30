package database

import (
	"sync"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	once sync.Once
	db   *gorm.DB
)

func MysqlInit(dsn string) *gorm.DB {
	var err error
	once.Do(func() {
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			panic(err)
		}
	})
	return db
}
func GetDB(dsn string) *gorm.DB {
	if db == nil {
		return MysqlInit(dsn)
	}
	return db
}
