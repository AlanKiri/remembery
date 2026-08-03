package main

import (
	"fmt"
	"log"

	"github.com/alankiri/password-memorizer-tui/internal/config"
	"github.com/alankiri/password-memorizer-tui/internal/levels"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	levelsList, err := levels.Load()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("passmem ready: audio=%v, levels=%d\n", cfg.Audio, len(levelsList))
}
