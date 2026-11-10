package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/D4nRusu/band4band/backend/internal/database"
	"github.com/D4nRusu/band4band/backend/internal/httpapi"
	"github.com/D4nRusu/band4band/backend/internal/services"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment")
	}

	ctx := context.Background()

	app, db, err := database.InitFirebase(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	authClient, err := app.Auth(ctx)
	if err != nil {
		log.Fatal(err)
	}

	var receiptExtractor httpapi.ReceiptExtractor
	if endpoint := os.Getenv("RECEIPT_LLM_URL"); endpoint != "" {
		extractor, err := services.NewReceiptExtractor(endpoint, os.Getenv("RECEIPT_LLM_MODEL"))
		if err != nil {
			log.Fatal(err)
		}

		receiptExtractor = extractor
	}

	log.Println("Listening on :8080")
	router := httpapi.NewRouter(
		authClient,
		database.NewUserStore(db),
		database.NewGroupStore(db),
		database.NewInvitationStore(db),
		receiptExtractor,
	)
	log.Fatal(http.ListenAndServe(":8080", router))
}
