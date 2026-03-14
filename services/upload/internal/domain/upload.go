package domain

import (
	"errors"
	"path/filepath"
	"strings"
)

var AllowedExtensions = []string{".mp4"}

var ErrInvalidExtension = errors.New("invalid file extension: only mp4 is allowed")

func ValidateUploadExtension(filename string) error {
	if filename == "" {
		return errors.New("filename is required")
	}
	ext := strings.ToLower(filepath.Ext(filename))
	for _, allowed := range AllowedExtensions {
		if ext == allowed {
			return nil
		}
	}
	return ErrInvalidExtension
}
