package model

import "time"

type NormalTimeSequenceData struct {
	Timestamp time.Time `json:"ts"`
	Id        string    `json:"id"`
	Value     float64   `json:"val"`
	Status    int       `json:"status"`
}

type STRTimeSequenceData struct {
	Timestamp time.Time `json:"ts"`
	Id        string    `json:"id"`
	Value     string    `json:"val"`
	Status    int       `json:"status"`
}
