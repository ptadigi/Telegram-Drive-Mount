package api

import (
	"net/url"
	"os"
)

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func urlQueryEscape(value string) string {
	return url.QueryEscape(value)
}
