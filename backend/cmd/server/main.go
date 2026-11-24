package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"rsi-surat/internal/config"
	"rsi-surat/internal/db"
	"rsi-surat/internal/handlers"
	mw "rsi-surat/internal/middleware"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("koneksi database gagal: ", err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatal("migrasi gagal: ", err)
	}
	if err := db.Seed(ctx, pool); err != nil {
		log.Fatal("seed gagal: ", err)
	}
	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		log.Fatal(err)
	}

	h := &handlers.Handler{DB: pool, Cfg: cfg}

	isActive := func(ctx context.Context, id int) bool {
		var ok bool
		if err := pool.QueryRow(ctx, `SELECT active FROM users WHERE id=$1`, id).Scan(&ok); err != nil {
			return false
		}
		return ok
	}
	auth := mw.Auth(cfg.JWTSecret, isActive)
	protect := func(f http.HandlerFunc, roles ...string) http.Handler {
		var hd http.Handler = f
		if len(roles) > 0 {
			hd = mw.RequireRole(roles...)(hd)
		}
		return auth(hd)
	}

	mux := http.NewServeMux()

	// publik
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("GET /api/verify/{code}", h.Verify)
	mux.Handle("GET /api/me", protect(h.Me))

	// mahasiswa
	mux.Handle("GET /api/letter-types", protect(h.ListLetterTypes, "mahasiswa", "staf", "dosen"))
	mux.Handle("GET /api/letter-types/{id}", protect(h.GetLetterType, "mahasiswa", "staf", "dosen"))
	mux.Handle("POST /api/tickets", protect(h.CreateTicket, "mahasiswa"))
	mux.Handle("GET /api/tickets", protect(h.MyTickets, "mahasiswa"))
	mux.Handle("POST /api/tickets/{id}/resubmit", protect(h.Resubmit, "mahasiswa"))
	mux.Handle("GET /api/archive", protect(h.Archive, "mahasiswa"))

	// bersama
	mux.Handle("GET /api/tickets/{id}", protect(h.GetTicket, "mahasiswa", "staf", "dosen"))
	mux.Handle("GET /api/tickets/{id}/letter", protect(h.LetterFile, "mahasiswa", "staf", "dosen"))
	mux.Handle("GET /api/documents/{id}/file", protect(h.DocumentFile, "mahasiswa", "staf", "dosen"))
	mux.Handle("GET /api/messages", protect(h.ListMessages, "mahasiswa", "staf"))
	mux.Handle("POST /api/messages", protect(h.SendMessage, "mahasiswa", "staf"))

	// dosen (US-09)
	mux.Handle("GET /api/dosen/requests", protect(h.DosenList, "dosen"))
	mux.Handle("GET /api/dosen/requests/{id}", protect(h.DosenDetail, "dosen"))
	mux.Handle("POST /api/dosen/requests/{id}/decision", protect(h.DosenDecision, "dosen"))

	// staf: verifikasi (US-10, 11, 12)
	mux.Handle("GET /api/staff/tickets", protect(h.StaffTickets, "staf"))
	mux.Handle("POST /api/staff/tickets/{id}/start", protect(h.StartReview, "staf"))
	mux.Handle("PATCH /api/staff/tickets/{id}/status", protect(h.UpdateStatus, "staf"))
	mux.Handle("POST /api/staff/tickets/{id}/issue", protect(h.IssueLetter, "staf"))
	mux.Handle("GET /api/staff/conversations", protect(h.Conversations, "staf"))

	// staf: layanan & template (US-13)
	mux.Handle("GET /api/staff/letter-types", protect(h.AdminListLetters, "staf"))
	mux.Handle("GET /api/staff/letter-types/{id}", protect(h.AdminGetLetter, "staf"))
	mux.Handle("POST /api/staff/letter-types", protect(h.AdminCreateLetter, "staf"))
	mux.Handle("PUT /api/staff/letter-types/{id}", protect(h.AdminUpdateLetter, "staf"))
	mux.Handle("GET /api/staff/settings", protect(h.GetSettings, "staf"))
	mux.Handle("PUT /api/staff/settings", protect(h.PutSettings, "staf"))

	// staf: hak akses & audit (US-14)
	mux.Handle("GET /api/staff/users", protect(h.AdminUsers, "staf"))
	mux.Handle("PATCH /api/staff/users/{id}/access", protect(h.AdminSetAccess, "staf"))
	mux.Handle("GET /api/staff/audit", protect(h.AuditLogs, "staf"))

	// staf: laporan (US-15)
	mux.Handle("GET /api/staff/reports", protect(h.Report, "staf"))
	mux.Handle("GET /api/staff/reports/export", protect(h.ReportExport, "staf"))

	log.Printf("API berjalan di :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, mw.CORS(cfg.FrontendOrigin, mux)))
}
