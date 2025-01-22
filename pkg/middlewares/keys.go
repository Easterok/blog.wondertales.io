package middlewares

type Key int

const (
	ContextLangKey Key = iota
	ContextStaticHashKey
	ContextResolvedHost

	ContextPreferLanguage
	ContextBaseHref
	ContextFullpath
	ContextHost
)
