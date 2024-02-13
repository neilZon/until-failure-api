package main

import (
	"fmt"
	"log"
	"os"

	"github.com/clerkinc/clerk-sdk-go/clerk"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
	db "github.com/neilZon/workout-logger-api/database"
	"github.com/neilZon/workout-logger-api/middleware"

	"github.com/neilZon/workout-logger-api/handlers"
)

const (
	defaultPort   = "3000"
	publicKeyPath = "public.pem"
)

func main() {
	environment := os.Getenv("ENVIRONMENT")

	if environment != "PROD" {
		err := godotenv.Load()
		if err != nil {
			log.Fatalf("Error loading .env file")
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	db, err := db.InitDb()
	if err != nil {
		log.Fatal(err)
	}

	clerk_secret := os.Getenv("CLERK_SECRET")
	client, err := clerk.NewClient(clerk_secret)
	if err != nil {
		log.Fatal(err)
	}

	app := fiber.New()

	app.Use(recover.New())

	validate := validator.New(validator.WithRequiredStructEnabled())

	h := handlers.Handler{DB: db, Validate: validate}

	m := middleware.Middleware{Clerk: client}

	handlers.RegisterRoutes(app, h, m)

	addr := fmt.Sprintf("0.0.0.0:%s", port)

	app.Listen(addr)
}
