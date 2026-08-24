package main

import (
	"fmt"

	"agenthub/internal/config"
)

func main() {
	fmt.Println("AgentHub starting...")

	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		fmt.Println("load config:", err)
		return
	}

	fmt.Printf("name=%s env=%s address=%s:%d\n", cfg.App.Name, cfg.App.Env, cfg.Server.Host, cfg.Server.Port)
}
