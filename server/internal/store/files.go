package store

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"time"
)

type FileStore struct {
	baseDir string
}

func NewFileStore(baseDir string) *FileStore {
	return &FileStore{baseDir: baseDir}
}

func (f *FileStore) Save(clientID string, capturedAt time.Time, monitor int, r io.Reader) (relativePath string, sha string, size int64, err error) {
	datePath := capturedAt.UTC().Format("2006/01/02")
	dir := filepath.Join(f.baseDir, "screenshots", clientID, datePath)
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return "", "", 0, err
	}

	filename := capturedAt.UTC().Format("150405") + "-" + time.Now().UTC().Format("000000000") + "-m" + itoa(monitor) + ".jpg"
	absolute := filepath.Join(dir, filename)
	relativePath, err = filepath.Rel(f.baseDir, absolute)
	if err != nil {
		return "", "", 0, err
	}

	file, err := os.Create(absolute)
	if err != nil {
		return "", "", 0, err
	}
	defer file.Close()

	hash := sha256.New()
	w := io.MultiWriter(file, hash)
	n, err := io.Copy(w, r)
	if err != nil {
		return "", "", 0, err
	}
	return filepath.ToSlash(relativePath), hex.EncodeToString(hash.Sum(nil)), n, nil
}

func (f *FileStore) Delete(rel string) error {
	return os.Remove(filepath.Join(f.baseDir, rel))
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	buf := make([]byte, 0, 12)
	for v > 0 {
		buf = append([]byte{byte('0' + (v % 10))}, buf...)
		v /= 10
	}
	return sign + string(buf)
}
