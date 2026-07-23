package ginadapter_test

import (
	"testing"

	"github.com/gin-gonic/gin"
	gouigin "github.com/zatrano/goui/adapters/gin"
	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/i18n"
	"github.com/zatrano/goui/ws"
)

func TestRegister_DoesNotPanic(_ *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	server := ws.NewServer(ws.NewHub(), core.NewRegistry(), i18n.NewTranslator())
	defer server.Hub.Stop()
	gouigin.Register(r, gouigin.Options{Server: server})
}
