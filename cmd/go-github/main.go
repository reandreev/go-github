package main

import (
	"log"

	"github.com/reandreev/go-github/internal/routes"
)

func main() {
	err := routes.InitRouter(true).Run(":8080")
	if err != nil {
		log.Fatal(err)
	}
}
