package domain

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UserStatus string

const (
	UserStatusCreated  UserStatus = "created"
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
)

type OrderStatus string

const (
	OrderStatusSuccess OrderStatus = "success"
	OrderStatusFailed  OrderStatus = "failed"
)

type User struct {
	ID        string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	Contact   string     `gorm:"type:varchar(20)" json:"contact"`
	Email     string     `gorm:"type:varchar(255)" json:"email"`
	Status    UserStatus `gorm:"type:varchar(20)" json:"status"`
	CreatedAt int64      `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt int64      `gorm:"autoUpdateTime:milli" json:"updated_at"`
}

type Product struct {
	ID          string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	Description string         `gorm:"type:varchar(500)" json:"description"`
	Brand       string         `gorm:"type:varchar(100)" json:"brand"`
	Amount      int64          `gorm:"type:bigint" json:"amount"`
	Quantity    int            `gorm:"type:int" json:"quantity"`
	Metadata    datatypes.JSON `gorm:"type:json" json:"metadata"`
	CreatedAt   int64          `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt   int64          `gorm:"autoUpdateTime:milli" json:"updated_at"`
}

type Order struct {
	ID            string         `gorm:"type:varchar(14);primaryKey" json:"id"`
	UserID        string         `gorm:"type:varchar(36);index" json:"user_id"`
	ProductID     string         `gorm:"type:varchar(36)" json:"product_id"`
	Amount        int64          `gorm:"type:bigint" json:"amount"`
	Quantity      int            `gorm:"type:int" json:"quantity"`
	Status        OrderStatus    `gorm:"type:varchar(20)" json:"status"`
	Metadata      datatypes.JSON `gorm:"type:json" json:"metadata,omitempty"`
	FailureReason *string        `gorm:"type:varchar(500)" json:"failure_reason,omitempty"`
	RequestID     string         `gorm:"type:varchar(255);uniqueIndex" json:"request_id"`
	CreatedAt     int64          `gorm:"autoCreateTime:milli" json:"created_at"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	if u.CreatedAt == 0 {
		u.CreatedAt = time.Now().UnixMilli()
	}
	u.UpdatedAt = u.CreatedAt
	return nil
}

func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if p.CreatedAt == 0 {
		p.CreatedAt = time.Now().UnixMilli()
	}
	p.UpdatedAt = p.CreatedAt
	return nil
}

func GenerateOrderID() string {
	b := make([]byte, 7)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.ID == "" {
		o.ID = GenerateOrderID()
	}
	if o.CreatedAt == 0 {
		o.CreatedAt = time.Now().UnixMilli()
	}
	return nil
}
