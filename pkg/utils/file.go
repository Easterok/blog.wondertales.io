package utils

import (
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func ValidateFile(c echo.Context, name string, maxFile int64) (*multipart.FileHeader, error) {
	err := c.Request().ParseMultipartForm(10 << 20)

	if err != nil {
		return nil, c.String(http.StatusRequestEntityTooLarge, "File should be less than 10mb")
	}

	fileHeader, err := c.FormFile("file")

	if err != nil {
		return nil, c.String(http.StatusBadRequest, err.Error())
	}

	return fileHeader, nil
}

func FileExtention(name string) string {
	return strings.ToLower(filepath.Ext(name))
}

func FileExtentionWithoutDot(name string) string {
	return FileExtention(name)[1:]
}

func UUIDFileName(name string) string {
	return uuid.NewString() + FileExtention(name)
}
