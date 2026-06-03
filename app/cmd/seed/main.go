package main

import (
	"flag"
	"fmt"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/auth"
)

func main() {
	pepper := flag.String("pepper", "movie-suggestion-123456", "pepper for Argon2id")
	password := flag.String("password", "", "password to hash")
	flag.Parse()
	if *password == "" {
		fmt.Println("Usage: go run ./cmd/seed -password <password> [-pepper <pepper>]")
		return
	}
	svc := auth.NewPasswordService(*pepper)
	hash, err := svc.Hash(*password)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println(hash)
}
