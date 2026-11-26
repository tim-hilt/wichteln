package main

import (
	"crypto/rand"
	"log"
	"net/http"
	"os"
)

func main() {
	addr := env("LISTEN_ADDR", ":8080")
	dataDir := env("DATA_DIR", "data")
	baseURL := env("BASE_URL", "http://localhost:8080")

	mailer, err := mailerFromEnv(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	handler := NewApp(dataDir, mailer, baseURL, rand.Reader)
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
