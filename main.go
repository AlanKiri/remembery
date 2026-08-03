package main

import (
	"log"

	"github.com/alankiri/password-memorizer-tui/internal/tui"
)

func main() {
	if err := tui.Run(); err != nil {
		log.Fatal(err)
	}
}
