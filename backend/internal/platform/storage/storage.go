// Package storage provides local-disk file storage for uploaded attachments
// (receipt images, request photos, announcement attachments).
package storage

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// DefaultMaxSize is the per-file upload limit (5 MB).
const DefaultMaxSize = 5 << 20 // 5 MiB

var (
	// ErrUnsupportedType is returned for non-image/non-PDF uploads.
	ErrUnsupportedType = errors.New("unsupported file type")
	// ErrTooLarge is returned when a file exceeds the size limit.
	ErrTooLarge = errors.New("file too large")
)

// Service stores files on local disk under a root directory.
type Service struct {
	Root    string
	MaxSize int64
}

// New returns a disk-backed storage service. A non-positive maxSize uses
// DefaultMaxSize.
func New(root string, maxSize int64) *Service {
	if maxSize <= 0 {
		maxSize = DefaultMaxSize
	}
	return &Service{Root: root, MaxSize: maxSize}
}

// Save writes the stream to disk under a UUID-derived filename and returns the
// generated id, the path relative to Root, the detected content type, and the
// byte count. Only images and PDFs are accepted.
func (s *Service) Save(r io.Reader) (id uuid.UUID, relPath, contentType string, size int64, err error) {
	head := make([]byte, 512)
	n, readErr := io.ReadFull(r, head)
	if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
		return uuid.Nil, "", "", 0, fmt.Errorf("read file: %w", readErr)
	}
	head = head[:n]

	contentType = http.DetectContentType(head)
	if !allowed(contentType) {
		return uuid.Nil, "", "", 0, ErrUnsupportedType
	}

	if err := os.MkdirAll(s.Root, 0o755); err != nil {
		return uuid.Nil, "", "", 0, fmt.Errorf("create storage root: %w", err)
	}

	id = uuid.New()
	relPath = id.String() + extFor(contentType)
	abs := filepath.Join(s.Root, relPath)

	dst, err := os.Create(abs)
	if err != nil {
		return uuid.Nil, "", "", 0, fmt.Errorf("create file: %w", err)
	}

	src := io.MultiReader(bytes.NewReader(head), r)
	written, copyErr := io.Copy(dst, io.LimitReader(src, s.MaxSize+1))
	closeErr := dst.Close()
	if copyErr != nil {
		os.Remove(abs)
		return uuid.Nil, "", "", 0, fmt.Errorf("write file: %w", copyErr)
	}
	if closeErr != nil {
		os.Remove(abs)
		return uuid.Nil, "", "", 0, fmt.Errorf("close file: %w", closeErr)
	}
	if written > s.MaxSize {
		os.Remove(abs)
		return uuid.Nil, "", "", 0, ErrTooLarge
	}

	return id, relPath, contentType, written, nil
}

func allowed(contentType string) bool {
	return strings.HasPrefix(contentType, "image/") || contentType == "application/pdf"
}

func extFor(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "application/pdf":
		return ".pdf"
	default:
		return ""
	}
}
