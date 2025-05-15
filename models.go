package main

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	TelegramID int64
	PlanID     uint
	Plan       UserPlan
}

type PlanType int

func (s PlanType) String() string {
	switch s {
	case Monthly:
		return "monthly"
	case Yearly:
		return "yearly"
	}
	return "unknown"
}

const (
	Monthly PlanType = iota
	Yearly
)

type UserPlan struct {
	gorm.Model
	Type     PlanType
	IsActive bool
}

func (u UserPlan) IsExpired() bool {
	return time.Now().After(u.CreatedAt)
}

// func (u UserPlanBase)SetExpiresAt(){
// 	switch u.Type {
// 	case Monthly:
// 		u.expiresAt = time.Now().AddDate(0,1,0)
// 	case Yearly:
// 		u.expiresAt = time.Now().AddDate(1,0,0)
// 	}
// }
