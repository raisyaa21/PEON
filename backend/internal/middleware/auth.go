package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID int    `json:"uid"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type ctxKey struct{}

func jsonErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// isActive dipakai agar akun yang dinonaktifkan langsung ditolak walau token masih berlaku.
func Auth(secret string, isActive func(ctx context.Context, id int) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			tokenStr := strings.TrimPrefix(header, "Bearer ")
			if header == "" || tokenStr == header {
				jsonErr(w, http.StatusUnauthorized, "Token tidak ditemukan")
				return
			}
			claims := &Claims{}
			tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
				return []byte(secret), nil
			}, jwt.WithValidMethods([]string{"HS256"}))
			if err != nil || !tok.Valid {
				jsonErr(w, http.StatusUnauthorized, "Sesi tidak valid atau sudah berakhir")
				return
			}
			if isActive != nil && !isActive(r.Context(), claims.UserID) {
				jsonErr(w, http.StatusUnauthorized, "Akun Anda dinonaktifkan")
				return
			}
			ctx := context.WithValue(r.Context(), ctxKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func FromContext(ctx context.Context) *Claims {
	c, _ := ctx.Value(ctxKey{}).(*Claims)
	return c
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c := FromContext(r.Context())
			if c == nil || !slices.Contains(roles, c.Role) {
				jsonErr(w, http.StatusForbidden, "Anda tidak memiliki akses")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func CORS(origin string, next http.Handler) http.Handler {
	allowed := map[string]bool{
		origin:                  true,
		"http://localhost:3000": true,
		"http://127.0.0.1:3000": true,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o := r.Header.Get("Origin")
		if allowed[o] || strings.HasPrefix(o, "http://localhost:") || strings.HasPrefix(o, "http://127.0.0.1:") {
			w.Header().Set("Access-Control-Allow-Origin", o)
			w.Header().Set("Vary", "Origin")
		} else if o == "" && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
