package ginadapter_test

import (
	"testing"

	"github.com/gin-gonic/gin"
	gouigin "github.com/zatrano/goui/adapters/gin"
	"github.com/zatrano/goui/v2/core"
	"github.com/zatrano/goui/v2/i18n"
	"github.com/zatrano/goui/v2/ws"
)

func TestRegister_DoesNotPanic(_ *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	server := ws.NewServer(ws.NewHub(), core.NewRegistry(), i18n.NewTranslator())
	defer server.Hub.Stop()
	gouigin.Register(r, gouigin.Options{Server: server})
}
