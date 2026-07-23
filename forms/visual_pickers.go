package forms

// EmojiItems curated emoji set for SearchableSelect.
func EmojiItems() []SelectItem {
	return []SelectItem{
		{Value: "😀", Label: "😀 gülümseme"},
		{Value: "😂", Label: "😂 kahkaha"},
		{Value: "😍", Label: "😍 aşk"},
		{Value: "👍", Label: "👍 beğeni"},
		{Value: "🔥", Label: "🔥 ateş"},
		{Value: "✅", Label: "✅ onay"},
		{Value: "🎉", Label: "🎉 kutlama"},
		{Value: "🚀", Label: "🚀 roket"},
		{Value: "💡", Label: "💡 fikir"},
		{Value: "❤️", Label: "❤️ kalp"},
	}
}

// IconItems simple symbol icons (no icon font dependency).
func IconItems() []SelectItem {
	return []SelectItem{
		{Value: "★", Label: "★ star"},
		{Value: "●", Label: "● circle"},
		{Value: "■", Label: "■ square"},
		{Value: "▲", Label: "▲ triangle"},
		{Value: "◆", Label: "◆ diamond"},
		{Value: "✓", Label: "✓ check"},
		{Value: "✕", Label: "✕ close"},
		{Value: "⚙", Label: "⚙ settings"},
		{Value: "⌂", Label: "⌂ home"},
		{Value: "✉", Label: "✉ mail"},
	}
}

// FontItems common web-safe / system font stacks.
func FontItems() []SelectItem {
	return []SelectItem{
		{Value: "system-ui, sans-serif", Label: "System UI"},
		{Value: "Georgia, serif", Label: "Georgia"},
		{Value: "\"Times New Roman\", Times, serif", Label: "Times New Roman"},
		{Value: "Arial, Helvetica, sans-serif", Label: "Arial"},
		{Value: "\"Courier New\", Courier, monospace", Label: "Courier New"},
		{Value: "Verdana, Geneva, sans-serif", Label: "Verdana"},
		{Value: "\"Trebuchet MS\", sans-serif", Label: "Trebuchet MS"},
		{Value: "Impact, Charcoal, sans-serif", Label: "Impact"},
	}
}

func NewEmojiPicker(name, event string) SearchableSelect {
	return newPicker(name, event, "Emoji seçin", EmojiItems())
}

func NewIconPicker(name, event string) SearchableSelect {
	return newPicker(name, event, "İkon seçin", IconItems())
}

func NewFontPicker(name, event string) SearchableSelect {
	return newPicker(name, event, "Font seçin", FontItems())
}

// MentionUsers sample @mention directory.
func MentionUsers() []SelectItem {
	return []SelectItem{
		{Value: "ayse", Label: "Ayşe Yılmaz"},
		{Value: "mehmet", Label: "Mehmet Demir"},
		{Value: "zeynep", Label: "Zeynep Kaya"},
		{Value: "ali", Label: "Ali Çelik"},
		{Value: "fatma", Label: "Fatma Arslan"},
	}
}
