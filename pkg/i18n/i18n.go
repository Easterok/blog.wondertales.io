package i18n

func TranslateBetween(lang, eng, ru string) string {
	if lang == "ru" {
		return ru
	}

	return eng
}

func Translate(lang, key string) string {
	if lang == "ru" {
		return RuLang[key]
	}

	return EnLang[key]
}
