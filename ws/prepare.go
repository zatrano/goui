package ws

import (
	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/i18n"
)

// PrepareComponent injects translator / BaseComponent ID+locale (same as a
// live session) so HTTP SSR can Mount+Render outside a WebSocket session.
func PrepareComponent(c core.Component, id, locale string, translator *i18n.Translator) {
	applySessionContext(c, id, locale, translator)
}

// DecorateHTML stamps data-goui-component on the root element (exported for SSR).
func DecorateHTML(html, componentID string) (string, error) {
	return decorateComponentHTML(html, componentID)
}
