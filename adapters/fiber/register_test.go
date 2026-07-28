package fiber_test

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	gouifiber "github.com/zatrano/goui/adapters/fiber"
	"github.com/zatrano/goui/v2/core"
	"github.com/zatrano/goui/v2/i18n"
	"github.com/zatrano/goui/v2/ws"
)

func TestRegister_DoesNotPanic(_ *testing.T) {
	app := fiber.New()
	server := ws.NewServer(ws.NewHub(), core.NewRegistry(), i18n.NewTranslator())
	defer server.Hub.Stop()
	gouifiber.Register(app, gouifiber.Options{Server: server})
}
