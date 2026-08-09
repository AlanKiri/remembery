package main

import (
	"log"

	"github.com/alankiri/password-memorizer-tui/internal/ui"
)

func main() {
	if err := ui.Run(); err != nil {
		log.Fatal(err)
	}
}
