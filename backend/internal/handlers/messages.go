package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"rsi-surat/internal/middleware"
)

type Message struct {
	ID         int       `json:"id"`
	SenderID   int       `json:"sender_id"`
	SenderName string    `json:"sender_name"`
	SenderRole string    `json:"sender_role"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
}

// mahasiswa selalu memakai thread miliknya; staf wajib menyebut student_id
func threadOwner(r *http.Request, fromBody int) int {
	c := middleware.FromContext(r.Context())
	if c.Role == "mahasiswa" {
		return c.UserID
	}
	if fromBody != 0 {
		return fromBody
	}
	id, _ := strconv.Atoi(r.URL.Query().Get("student_id"))
	return id
}

// US-06: riwayat pesan
func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	sid := threadOwner(r, 0)
	if sid == 0 {
		writeErr(w, http.StatusBadRequest, "student_id wajib diisi")
		return
	}
	rows, err := h.DB.Query(r.Context(), `
		SELECT m.id, m.sender_id, u.name, m.sender_role, m.body, m.created_at
		FROM messages m JOIN users u ON u.id = m.sender_id
		WHERE m.student_id=$1 ORDER BY m.created_at, m.id`, sid)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memuat pesan")
		return
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.SenderID, &m.SenderName, &m.SenderRole, &m.Body, &m.CreatedAt); err == nil {
			out = append(out, m)
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// US-06: kirim pesan (log dua arah tersimpan di tabel messages)
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	var in struct {
		Body      string `json:"body"`
		StudentID int    `json:"student_id"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "Data tidak valid")
		return
	}
	body := strings.TrimSpace(in.Body)
	if body == "" || len(body) > 2000 {
		writeErr(w, http.StatusBadRequest, "Pesan wajib diisi (maksimal 2000 karakter)")
		return
	}
	sid := threadOwner(r, in.StudentID)
	if sid == 0 {
		writeErr(w, http.StatusBadRequest, "student_id wajib diisi")
		return
	}
	if _, err := h.DB.Exec(r.Context(),
		`INSERT INTO messages (student_id, sender_id, sender_role, body) VALUES ($1,$2,$3,$4)`,
		sid, c.UserID, c.Role, body); err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal mengirim pesan")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "terkirim"})
}

// daftar percakapan untuk staf
func (h *Handler) Conversations(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `
		SELECT u.id, u.identifier, u.name, m.body, m.created_at
		FROM users u
		JOIN LATERAL (SELECT body, created_at FROM messages WHERE student_id=u.id ORDER BY created_at DESC, id DESC LIMIT 1) m ON true
		ORDER BY m.created_at DESC`)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memuat percakapan")
		return
	}
	defer rows.Close()
	type conv struct {
		StudentID  int       `json:"student_id"`
		Identifier string    `json:"identifier"`
		Name       string    `json:"name"`
		LastBody   string    `json:"last_body"`
		LastAt     time.Time `json:"last_at"`
	}
	out := []conv{}
	for rows.Next() {
		var c conv
		if err := rows.Scan(&c.StudentID, &c.Identifier, &c.Name, &c.LastBody, &c.LastAt); err == nil {
			out = append(out, c)
		}
	}
	writeJSON(w, http.StatusOK, out)
}
