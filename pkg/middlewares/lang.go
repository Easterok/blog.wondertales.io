package middlewares

import (
	"context"
	"strings"

	"github.com/labstack/echo/v4"
)

type PreferLanguageConfig struct {
	Skipper func(c echo.Context) bool

	Fallback string
	Accept   []string
}

var defaultPreferLanguageConfig = PreferLanguageConfig{
	Skipper: func(c echo.Context) bool {
		return c.Request().Method != "GET"
	},
	Fallback: "en",
	Accept:   []string{"en"},
}

func setLanguageAndGoNext(c echo.Context, next echo.HandlerFunc, lang string, setLinkCtx bool, host string) error {
	ctx := context.WithValue(c.Request().Context(), ContextPreferLanguage, lang)
	ctx = context.WithValue(ctx, ContextFullpath, c.Request().URL.Path)
	ctx = context.WithValue(ctx, ContextHost, host)

	if setLinkCtx {
		ctx = context.WithValue(ctx, ContextBaseHref, lang)
	}

	c.SetRequest(c.Request().WithContext(ctx))

	return next(c)
}

func resolveHost(s string) string {
	if strings.HasPrefix(s, "blog.wonder-tales.ru") {
		return "https://blog.wonder-tales.ru"
	}

	if strings.HasPrefix(s, "www.blog.wonder-tales.ru") {
		return "https://www.blog.wonder-tales.ru"
	}

	if strings.HasPrefix(s, "www.blog.wondertales.io") {
		return "https://www.blog.wondertales.io"
	}

	return "https://blog.wondertales.io"
}

func PreferLanguageWithConfig(cfg PreferLanguageConfig) echo.MiddlewareFunc {
	if cfg.Skipper == nil {
		cfg.Skipper = defaultPreferLanguageConfig.Skipper
	}

	if cfg.Fallback == "" {
		cfg.Fallback = defaultPreferLanguageConfig.Fallback
	}

	if cfg.Accept == nil {
		cfg.Accept = defaultPreferLanguageConfig.Accept
	}

	dict := make(map[string]bool, len(cfg.Accept))

	for _, l := range cfg.Accept {
		dict[l] = true
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if cfg.Skipper(c) {
				return next(c)
			}

			path := c.Request().URL.Path
			host := resolveHost(c.Request().Header.Get("X-Forwarded-Host"))

			for _, l := range cfg.Accept {
				if strings.HasPrefix(path, "/"+l+"/") || strings.HasSuffix(path, "/"+l) {
					return setLanguageAndGoNext(c, next, l, true, host)
				}
			}

			lang := cfg.Fallback

			if strings.HasSuffix(host, ".ru") {
				lang = "ru"
			}

			return setLanguageAndGoNext(c, next, lang, false, host)
		}
	}
}

func GetContextPreferLanguage(ctx context.Context) string {
	lang, ok := ctx.Value(ContextPreferLanguage).(string)

	if !ok {
		return "en"
	}

	return lang
}

func GetContextBaseHref(ctx context.Context) string {
	href, ok := ctx.Value(ContextBaseHref).(string)

	if !ok {
		return ""
	}

	return href
}

func GetContextFullpath(ctx context.Context) string {
	path, ok := ctx.Value(ContextFullpath).(string)

	if !ok {
		return "/"
	}

	return path
}

func GetContextHost(ctx context.Context) string {
	host, ok := ctx.Value(ContextHost).(string)

	if !ok {
		return ""
	}

	return host
}
