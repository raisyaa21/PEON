package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Port           string
	DatabaseURL    string
	JWTSecret      string
	UploadDir      string
	FrontendOrigin string
	DeptCode       string
}

func Load() Config {
	uploadDir := get("UPLOAD_DIR", "./uploads")
	if abs, err := filepath.Abs(uploadDir); err == nil {
		uploadDir = abs
	}
	return Config{
		Port:           get("PORT", "8080"),
		DatabaseURL:    get("DATABASE_URL", "postgres://rsi:rsi123@localhost:5432/rsi_surat?sslmode=disable"),
		JWTSecret:      get("JWT_SECRET", "ganti-secret-ini-di-produksi"),
		UploadDir:      uploadDir,
		FrontendOrigin: get("FRONTEND_ORIGIN", "http://localhost:3000"),
		DeptCode:       get("DEPT_CODE", "SI"),
	}
}

func get(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
