package main

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func init() {
	dsn := fmt.Sprintf("host=db user=postgres password=%s dbname=postgres port=5432 sslmode=disable TimeZone=Asia/Shanghai", os.Getenv("POSTGRES_PASSWORD"))
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logrus.Error(err)
	}
	err = db.AutoMigrate(&User{})
	if err != nil {
		logrus.Error(err)
	}
	err = db.AutoMigrate(&UserPlan{})
	if err != nil {
		logrus.Error(err)
	}
	DB = db
}

func UserGetByTelegramID(id int64) (*User, error) {
	var user *User
	result := DB.First(&user, "telegram_id = ?", id)
	return user, result.Error
}

func PlansGetExpired() ([]UserPlan, error) {
	plans := make([]UserPlan, 0)
	result := DB.Where("expires_at >= ?", time.Now()).Find(&plans)
	return plans, result.Error
}
