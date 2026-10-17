package main

import (
	"context"
	"example.com/hackathon/apps/api/internal/database"
	"log"
	"os"
	"time"
)

func main() {
	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	if command != "up" && command != "down" && command != "status" {
		log.Fatal("Укажите команду up, down или status")
	}
	if os.Getenv("DATABASE_URL") == "" {
		log.Fatal("Укажите DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := database.Migrate(ctx, os.Getenv("DATABASE_URL"), command); err != nil {
		log.Fatal(err)
	}
}
