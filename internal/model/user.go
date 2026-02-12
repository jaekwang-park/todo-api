package model

import "time"

type User struct {
	ID              string    `json:"id"`
	CognitoSub      string    `json:"cognito_sub"`
	Email           string    `json:"email"`
	Nickname        string    `json:"nickname"`
	ProfileImageURL string    `json:"profile_image_url"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
