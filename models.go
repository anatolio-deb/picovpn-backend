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
	Type      PlanType
	IsActive  bool
	ExpiresAt time.Time
}
