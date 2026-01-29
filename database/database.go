package database

import (
	"os"
)

func Conn() string {
	return os.Getenv("DATABASE_URL")
}
