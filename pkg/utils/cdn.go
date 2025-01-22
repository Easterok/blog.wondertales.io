package utils

import (
	"fmt"
	"os"
	"strings"
)

func Cdn(s string) string {
	if s == "" || strings.HasPrefix(s, "/") {
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com%s", os.Getenv("AWS_BUCKET"), os.Getenv("AWS_REGION"), s)
	}

	return s
}
