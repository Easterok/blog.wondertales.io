package utils

import "github.com/labstack/echo/v4"

func Htmx(c echo.Context) bool {
	return c.Request().Header.Get("HX-Request") == "true"
}
