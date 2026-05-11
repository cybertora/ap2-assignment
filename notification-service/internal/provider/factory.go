package provider

import (
	"log"
	"strings"

	"notification-service/internal/app"
	"notification-service/internal/domain"
)

// New — фабрика провайдеров по env PROVIDER_MODE.
// Это и есть точка переключения REAL / SIMULATED.
func New(cfg *app.Config) domain.EmailSender {
	switch strings.ToUpper(cfg.ProviderMode) {
	case "REAL":
		log.Printf("[PROVIDER] mode=REAL host=%s", cfg.SMTPHost)
		return NewSMTPSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom)
	default:
		log.Println("[PROVIDER] mode=SIMULATED (random failures + latency)")
		return NewSimulatedSender()
	}
}
