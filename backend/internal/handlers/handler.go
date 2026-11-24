package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"rsi-surat/internal/config"
	"rsi-surat/internal/middleware"
)

type Handler struct {
	DB  *pgxpool.Pool
	Cfg config.Config
}

// dbq dipenuhi oleh pgxpool.Pool maupun pgx.Tx
type dbq interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func decode(r *http.Request, v any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(v)
}

// audit mencatat aksi penting ke audit_logs (US-14)
func (h *Handler) audit(ctx context.Context, actorID int, action string) {
	var name string
	_ = h.DB.QueryRow(ctx, `SELECT name FROM users WHERE id=$1`, actorID).Scan(&name)
	_, _ = h.DB.Exec(ctx, `INSERT INTO audit_logs (actor_id, actor_name, action) VALUES ($1,$2,$3)`, actorID, name, action)
}

type User struct {
	ID         int    `json:"id"`
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	Position   string `json:"position"`
}

const loginErrMsg = "Kombinasi Nomor Identitas (NIM/NIP) atau Kata Sandi salah, atau akun belum terdaftar. Silakan periksa kembali data Anda."

// US-01
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Identifier string `json:"identifier"`
		Password   string `json:"password"`
		Type       string `json:"type"`
	}
	if err := decode(r, &in); err != nil || strings.TrimSpace(in.Identifier) == "" || in.Password == "" {
		writeErr(w, http.StatusBadRequest, "NIM/NIP dan kata sandi wajib diisi")
		return
	}

	var u User
	var hash string
	var active bool
	err := h.DB.QueryRow(r.Context(),
		`SELECT id, identifier, name, role, position, active, password_hash FROM users WHERE identifier=$1`,
		strings.TrimSpace(in.Identifier)).Scan(&u.ID, &u.Identifier, &u.Name, &u.Role, &u.Position, &active, &hash)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		writeErr(w, http.StatusUnauthorized, loginErrMsg)
		return
	}
	if !active {
		writeErr(w, http.StatusForbidden, "Akun Anda dinonaktifkan. Hubungi staf administrasi akademik.")
		return
	}

	isStudent := u.Role == "mahasiswa"
	if (in.Type == "mahasiswa" && !isStudent) || (in.Type == "pegawai" && isStudent) {
		writeErr(w, http.StatusUnauthorized, loginErrMsg)
		return
	}

	claims := &middleware.Claims{
		UserID: u.ID,
		Role:   u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(8 * time.Hour)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(h.Cfg.JWTSecret))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal membuat sesi")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": u})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	var u User
	err := h.DB.QueryRow(r.Context(),
		`SELECT id, identifier, name, role, position FROM users WHERE id=$1`, c.UserID).
		Scan(&u.ID, &u.Identifier, &u.Name, &u.Role, &u.Position)
	if err != nil {
		writeErr(w, http.StatusNotFound, "Pengguna tidak ditemukan")
		return
	}
	writeJSON(w, http.StatusOK, u)
}
