package main

import (
	"fmt"
	"weekly-shopping-app/database"

	"github.com/joho/godotenv"
)

func main() {
	loadEnv()

	fmt.Printf("%v", database.Conn())
}

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}
}
