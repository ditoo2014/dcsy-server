package model

// WeakPassword stores one plaintext entry from the weak-password dictionary.
type WeakPassword struct {
	ID       uint64 `gorm:"primaryKey;column:id"`
	Password string `gorm:"column:password;type:varchar(255);not null"`
}

func (WeakPassword) TableName() string {
	return "public.weak_password_dictionary"
}
