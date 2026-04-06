package infrastructure

import "time"

type contactModel struct {
	ID         string `gorm:"primaryKey;column:id;type:char(36)"`
	TenantID   string `gorm:"column:tenant_id;type:char(36);not null"`
	Name       string `gorm:"column:name;type:varchar(255);not null;default:''"`
	Phone      string `gorm:"column:phone;type:varchar(20);not null"`
	WhatsAppID string `gorm:"column:whatsapp_id;type:varchar(100);not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (contactModel) TableName() string { return "contacts" }

type conversationModel struct {
	ID            string `gorm:"primaryKey;column:id;type:char(36)"`
	TenantID      string `gorm:"column:tenant_id;type:char(36);not null"`
	ContactID     string `gorm:"column:contact_id;type:char(36);not null"`
	Status        string `gorm:"column:status;type:enum('open','closed');not null;default:'open'"`
	LastMessageAt time.Time
	UnreadCount   int `gorm:"column:unread_count;type:int;not null;default:0"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (conversationModel) TableName() string { return "conversations" }

type messageModel struct {
	ID             string `gorm:"primaryKey;column:id;type:char(36)"`
	ConversationID string `gorm:"column:conversation_id;type:char(36);not null"`
	Direction      string `gorm:"column:direction;type:enum('incoming','outgoing');not null"`
	Content        string `gorm:"column:content;type:text;not null"`
	Type           string `gorm:"column:type;type:enum('text');not null;default:'text'"`
	Status         string `gorm:"column:status;type:enum('pending','sent','failed');not null;default:'sent'"`
	WhatsAppMsgID  *string `gorm:"column:whatsapp_msg_id;type:varchar(100)"`
	Timestamp      time.Time
	CreatedAt      time.Time
}

func (messageModel) TableName() string { return "messages" }
