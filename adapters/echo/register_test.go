package echoadapter_test

import (
	"testing"

	"github.com/labstack/echo/v4"
	gouiecho "github.com/zatrano/goui/adapters/echo"
	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/i18n"
	"github.com/zatrano/goui/ws"
)

func TestRegister_DoesNotPanic(_ *testing.T) {
	e := echo.New()
	server := ws.NewServer(ws.NewHub(), core.NewRegistry(), i18n.NewTranslator())
	defer server.Hub.Stop()
	gouiecho.Register(e, gouiecho.Options{Server: server})
}
