package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/marcel-zisser/amazons-game-server/internal/server"
)

var secret = os.Getenv("JWT_SIGNING_KEY")

func main() {
	// 1. Open the file
	file, err := os.Open("configs/teams.txt")
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	// 2. Ensure the file gets closed when main() finishes executing
	defer file.Close()

	// 3. Create a new scanner wrapped around our file reader
	scanner := bufio.NewScanner(file)

	// 4. Scan through the file line by line
	for scanner.Scan() {
		team := scanner.Text() // Grabs the current line as a string
		validity := 7 * 24 * time.Hour

		token, err := server.GenerateJWT(team, validity)
		if err != nil {
			log.Fatalf("Error generating token: %v", err)
		}

		fmt.Println("=================================================================")
		fmt.Printf("🎫 JWT Issued for: %s\n", team)
		fmt.Println("=================================================================")
		fmt.Printf("Token String:\n%s\n\n", token)
		fmt.Println("gRPC Metadata Authorization Header Input:")
		fmt.Printf("Bearer %s\n", token)
		fmt.Println("=================================================================")
	}

	// 5. Check if the scanner encountered any errors during processing
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error encountered while scanning: %v", err)
	}
}
