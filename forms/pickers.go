package forms

// CountryItems returns a curated ISO country list for SearchableSelect.
func CountryItems() []SelectItem {
	return []SelectItem{
		{Value: "TR", Label: "Türkiye"},
		{Value: "DE", Label: "Almanya"},
		{Value: "US", Label: "United States"},
		{Value: "GB", Label: "United Kingdom"},
		{Value: "FR", Label: "Fransa"},
		{Value: "NL", Label: "Hollanda"},
		{Value: "AT", Label: "Avusturya"},
		{Value: "CH", Label: "İsviçre"},
		{Value: "AZ", Label: "Azerbaycan"},
		{Value: "SA", Label: "Suudi Arabistan"},
		{Value: "AE", Label: "Birleşik Arap Emirlikleri"},
		{Value: "JP", Label: "Japonya"},
		{Value: "CN", Label: "Çin"},
		{Value: "IN", Label: "Hindistan"},
		{Value: "BR", Label: "Brezilya"},
		{Value: "CA", Label: "Kanada"},
		{Value: "AU", Label: "Avustralya"},
		{Value: "IT", Label: "İtalya"},
		{Value: "ES", Label: "İspanya"},
		{Value: "SE", Label: "İsveç"},
	}
}

// LanguageItems returns common language codes.
func LanguageItems() []SelectItem {
	return []SelectItem{
		{Value: "tr", Label: "Türkçe"},
		{Value: "en", Label: "English"},
		{Value: "de", Label: "Deutsch"},
		{Value: "fr", Label: "Français"},
		{Value: "ar", Label: "العربية"},
		{Value: "es", Label: "Español"},
		{Value: "ru", Label: "Русский"},
		{Value: "zh", Label: "中文"},
		{Value: "ja", Label: "日本語"},
		{Value: "pt", Label: "Português"},
	}
}

// TimezoneItems returns common IANA time zones.
func TimezoneItems() []SelectItem {
	return []SelectItem{
		{Value: "Europe/Istanbul", Label: "Europe/Istanbul (UTC+3)"},
		{Value: "Europe/Berlin", Label: "Europe/Berlin"},
		{Value: "Europe/London", Label: "Europe/London"},
		{Value: "America/New_York", Label: "America/New_York"},
		{Value: "America/Los_Angeles", Label: "America/Los_Angeles"},
		{Value: "Asia/Dubai", Label: "Asia/Dubai"},
		{Value: "Asia/Tokyo", Label: "Asia/Tokyo"},
		{Value: "Asia/Shanghai", Label: "Asia/Shanghai"},
		{Value: "Australia/Sydney", Label: "Australia/Sydney"},
		{Value: "UTC", Label: "UTC"},
	}
}

// CurrencyItems returns common ISO currency codes.
func CurrencyItems() []SelectItem {
	return []SelectItem{
		{Value: "TRY", Label: "TRY — Türk Lirası"},
		{Value: "USD", Label: "USD — US Dollar"},
		{Value: "EUR", Label: "EUR — Euro"},
		{Value: "GBP", Label: "GBP — Pound Sterling"},
		{Value: "CHF", Label: "CHF — Swiss Franc"},
		{Value: "JPY", Label: "JPY — Yen"},
		{Value: "AED", Label: "AED — Dirham"},
		{Value: "SAR", Label: "SAR — Riyal"},
	}
}

// DialCodeItems maps country ISO → phone dial code for PhoneInput.
func DialCodeItems() []SelectItem {
	return []SelectItem{
		{Value: "+90", Label: "TR +90"},
		{Value: "+49", Label: "DE +49"},
		{Value: "+1", Label: "US/CA +1"},
		{Value: "+44", Label: "GB +44"},
		{Value: "+33", Label: "FR +33"},
		{Value: "+31", Label: "NL +31"},
		{Value: "+43", Label: "AT +43"},
		{Value: "+41", Label: "CH +41"},
		{Value: "+994", Label: "AZ +994"},
		{Value: "+966", Label: "SA +966"},
		{Value: "+971", Label: "AE +971"},
		{Value: "+81", Label: "JP +81"},
		{Value: "+86", Label: "CN +86"},
		{Value: "+91", Label: "IN +91"},
	}
}

// NewCountryPicker returns a SearchableSelect preloaded with CountryItems.
func NewCountryPicker(name, event string) SearchableSelect {
	return newPicker(name, event, "Ülke seçin", CountryItems())
}

// NewLanguagePicker returns a SearchableSelect preloaded with LanguageItems.
func NewLanguagePicker(name, event string) SearchableSelect {
	return newPicker(name, event, "Dil seçin", LanguageItems())
}

// NewTimezonePicker returns a SearchableSelect preloaded with TimezoneItems.
func NewTimezonePicker(name, event string) SearchableSelect {
	return newPicker(name, event, "Saat dilimi", TimezoneItems())
}

// NewCurrencyPicker returns a SearchableSelect preloaded with CurrencyItems.
func NewCurrencyPicker(name, event string) SearchableSelect {
	return newPicker(name, event, "Para birimi", CurrencyItems())
}

func newPicker(name, event, placeholder string, items []SelectItem) SearchableSelect {
	if event == "" {
		event = name
	}
	return SearchableSelect{
		BaseSelectField: BaseSelectField{
			CommonAttrs: CommonAttrs{Name: name, ID: name},
			Placeholder: placeholder,
			Items:       items,
		},
		EventName: event,
	}
}
