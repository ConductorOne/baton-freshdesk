package main

import (
	cfg "github.com/conductorone/baton-freshdesk/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/config"
)

func main() {
	config.Generate("freshdesk", cfg.Configuration)
}
