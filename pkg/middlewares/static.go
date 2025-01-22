package middlewares

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/easterok/blogs/pkg/utils"
	"github.com/labstack/echo/v4"
)

func StaticCache(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if strings.HasPrefix(c.Path(), "/static") {
			maxAge := time.Hour * 24 * 30

			c.Response().Header().Set("Cache-Control", fmt.Sprintf("public,max-age=%d", int(maxAge.Seconds())))
		} else if c.Request().Method == "GET" && (strings.HasPrefix(c.Path(), "/admin") || utils.Htmx(c)) {
			c.Response().Header().Set("Cache-Control", "public,max-age=0,must-revalidate,no-cache,no-store")
			c.Response().Header().Set("Pragma", "no-cache")
			c.Response().Header().Set("Expires", "0")
		} else if c.Request().Method == "GET" {
			c.Response().Header().Set("Cache-Control", "public,max-age=0,must-revalidate")
		}

		return next(c)
	}
}

type StaticHashConfig struct {
	Hash string
}

func StaticHashWithConfig(config StaticHashConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := context.WithValue(c.Request().Context(), ContextStaticHashKey, config.Hash)

			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

func GetContextStaticHash(ctx context.Context) string {
	hash, ok := ctx.Value(ContextStaticHashKey).(string)

	if !ok {
		return ""
	}

	return hash
}

func FormatStaticLink(s string, hash string) string {
	a := strings.SplitN(s, ".", 2)

	return fmt.Sprintf("%s_%s.%s", a[0], hash, a[1])
}

func FormatStaticLinkFromContext(s string, ctx context.Context) string {
	hash := GetContextStaticHash(ctx)

	return FormatStaticLink(s, hash)
}

func ServiceStaticWithHash(e *echo.Echo, links []string, hash string) {
	for _, link := range links {
		hashed := FormatStaticLink(link, hash)

		e.File(fmt.Sprintf("/%s", hashed), link)
	}
}
