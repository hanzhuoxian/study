package main

import (
	"gorm.io/gorm"
)

// Person base structure describe person's base info.
type Person struct {
	*gorm.Model
	Notes      []*Notes
	Name       string `gorm:"column:name" json:"name"`
	Gender     string `gorm:"column:gender" json:"gender"`
	Birth      string `gorm:"column:birth" json:"birth"`
	BirthLocal string `gorm:"column:birth_local" json:"birthLocal"`
	IDNumber   string `gorm:"column:id_number" json:"IDNumber"`
	BankCard   uint   `gorm:"column:bank_card" json:"bankCard"`
}

func (p Person) TableName() string {
	return "person"
}

type Notes struct {
	Notes    string `gorm:"column:notes" json:"notes"`
	PersonID uint
}

func (n Notes) TableName() string {
	return "notes"
}
