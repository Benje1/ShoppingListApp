package api

import (
	"weekly-shopping-app/authentication"
	"weekly-shopping-app/database"
)

func CreateUserService(input UserInput) error {
	hashed, err := authentication.HashPassword(input.Password)
	if err != nil {
		return err
	}
	return database.InsertUser(input.Name, input.Username, hashed)
}

func UpdateUserService(input UserInput) error {
	hashed, err := authentication.HashPassword(input.Password)
	if err != nil {
		return err
	}
	return UpdateUserInDB(input.Username, input.Name, hashed)
}
