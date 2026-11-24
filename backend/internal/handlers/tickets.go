package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"rsi-surat/internal/middleware"
)

const maxUpload = 5 << 20 // 5 MB

var statusUnit = map[string]string{
	"submitted":               "Sistem Persuratan Akademik",
	"awaiting_recommendation": "Dosen Pembimbing Akademik",
	"reviewing":               "Staf Administrasi Akademik",
	"revision":                "Staf Administrasi Akademik",
	"approving":               "Staf Administrasi Akademik",
	"completed":               "Staf Administrasi Akademik",
	"rejected":                "Staf Administrasi Akademik",
}

type Ticket struct {
	ID            int       `json:"id"`
	TicketNo      string    `json:"ticket_no"`
	LetterType    string    `json:"letter_type"`
	StudentName   string    `json:"student_name"`
	StudentNIM    string    `json:"student_nim"`
	StudentID     int       `json:"student_id"`
	Purpose       string    `json:"purpose"`
	Status        string    `json:"status"`
	Note          string    `json:"note"`
	VerifierUnit  string    `json:"verifier_unit"`
	LetterTypeID  int       `json:"letter_type_id"`
	AdvisorStatus string    `json:"advisor_status"`
	AdvisorNote   string    `json:"advisor_note"`
	LetterNo      string    `json:"letter_no"`
	HasFinal      bool      `json:"has_final"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Requirement struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Document struct {
	ID              int       `json:"id"`
	RequirementID   int       `json:"requirement_id"`
	RequirementName string    `json:"requirement_name"`
	FileName        string    `json:"file_name"`
	Mime            string    `json:"mime"`
	SizeBytes       int64     `json:"size_bytes"`
	Version         int       `json:"version"`
	CreatedAt       time.Time `json:"created_at"`
}

type History struct {
	ID        int       `json:"id"`
	Status    string    `json:"status"`
	Unit      string    `json:"unit"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

type TicketDetail struct {
	Ticket
	History      []History     `json:"history"`
	Documents    []Document    `json:"documents"`
	Requirements []Requirement `json:"requirements"`
}

type scanner interface{ Scan(dest ...any) error }

const ticketSelect = `
SELECT t.id, t.ticket_no, lt.name, u.name, u.identifier, t.student_id, t.purpose, t.status,
       t.note, t.verifier_unit, t.letter_type_id, t.advisor_status, t.advisor_note,
       COALESCE(l.letter_no,''), COALESCE(l.final_path <> '', false), t.created_at, t.updated_at
FROM tickets t
JOIN letter_types lt ON lt.id = t.letter_type_id
JOIN users u ON u.id = t.student_id
LEFT JOIN letters l ON l.ticket_id = t.id `

func scanTicket(s scanner) (Ticket, error) {
	var t Ticket
	err := s.Scan(&t.ID, &t.TicketNo, &t.LetterType, &t.StudentName, &t.StudentNIM, &t.StudentID,
		&t.Purpose, &t.Status, &t.Note, &t.VerifierUnit, &t.LetterTypeID, &t.AdvisorStatus, &t.AdvisorNote,
		&t.LetterNo, &t.HasFinal, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func (h *Handler) listTickets(ctx context.Context, where, order string, args ...any) ([]Ticket, error) {
	rows, err := h.DB.Query(ctx, ticketSelect+where+" ORDER BY "+order, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Ticket{}
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (h *Handler) requirements(ctx context.Context, letterID int) ([]Requirement, error) {
	rows, err := h.DB.Query(ctx,
		`SELECT id, name FROM letter_requirements WHERE letter_type_id=$1 ORDER BY id`, letterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Requirement{}
	for rows.Next() {
		var r Requirement
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (h *Handler) history(ctx context.Context, ticketID int) []History {
	out := []History{}
	rows, err := h.DB.Query(ctx,
		`SELECT id, status, unit, note, created_at FROM ticket_history WHERE ticket_id=$1 ORDER BY created_at, id`, ticketID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var x History
		if rows.Scan(&x.ID, &x.Status, &x.Unit, &x.Note, &x.CreatedAt) == nil {
			out = append(out, x)
		}
	}
	return out
}

func (h *Handler) documents(ctx context.Context, ticketID int) []Document {
	out := []Document{}
	rows, err := h.DB.Query(ctx, `
		SELECT d.id, d.requirement_id, lr.name, d.file_name, d.mime, d.size_bytes, d.version, d.created_at
		FROM ticket_documents d
		JOIN letter_requirements lr ON lr.id = d.requirement_id
		WHERE d.ticket_id=$1 AND d.is_current
		ORDER BY d.requirement_id`, ticketID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var d Document
		if rows.Scan(&d.ID, &d.RequirementID, &d.RequirementName, &d.FileName, &d.Mime, &d.SizeBytes, &d.Version, &d.CreatedAt) == nil {
			out = append(out, d)
		}
	}
	return out
}

func addHistory(ctx context.Context, tx pgx.Tx, ticketID int, status, note string) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO ticket_history (ticket_id, status, unit, note) VALUES ($1,$2,$3,$4)`,
		ticketID, status, statusUnit[status], note)
	return err
}

// dosen hanya boleh melihat tiket mahasiswa bimbingannya yang butuh rekomendasi
func (h *Handler) advises(ctx context.Context, dosenID, ticketID int) bool {
	var ok bool
	_ = h.DB.QueryRow(ctx, `
		SELECT EXISTS(
		  SELECT 1 FROM tickets t
		  JOIN users u ON u.id = t.student_id
		  JOIN letter_types lt ON lt.id = t.letter_type_id
		  WHERE t.id=$1 AND u.advisor_id=$2 AND lt.requires_recommendation)`, ticketID, dosenID).Scan(&ok)
	return ok
}

func (h *Handler) canView(ctx context.Context, c *middleware.Claims, t Ticket) bool {
	switch c.Role {
	case "staf":
		return true
	case "mahasiswa":
		return t.StudentID == c.UserID
	case "dosen":
		return h.advises(ctx, c.UserID, t.ID)
	}
	return false
}

// ---------- Validasi & penyimpanan file (US-03) ----------

type pendingFile struct {
	ReqID  int
	Header *multipart.FileHeader
	Ext    string
}

func validateUpload(fh *multipart.FileHeader) (string, error) {
	if fh.Size > maxUpload {
		return "", errors.New("ukuran file melebihi batas 5 MB")
	}
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if ext != ".pdf" && ext != ".jpg" && ext != ".jpeg" {
		return "", errors.New("format file harus PDF atau JPG")
	}
	f, err := fh.Open()
	if err != nil {
		return "", errors.New("file tidak dapat dibaca")
	}
	defer f.Close()
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	ct := http.DetectContentType(buf[:n])
	if (ext == ".pdf" && ct != "application/pdf") || (ext != ".pdf" && ct != "image/jpeg") {
		return "", errors.New("isi file tidak sesuai dengan ekstensinya")
	}
	return ext, nil
}

func (h *Handler) collectFiles(r *http.Request, reqs []Requirement, requireAll bool) ([]pendingFile, error) {
	var out []pendingFile
	for _, rq := range reqs {
		fhs := r.MultipartForm.File["req_"+strconv.Itoa(rq.ID)]
		if len(fhs) == 0 {
			if requireAll {
				return nil, fmt.Errorf("dokumen \"%s\" wajib diunggah", rq.Name)
			}
			continue
		}
		ext, err := validateUpload(fhs[0])
		if err != nil {
			return nil, fmt.Errorf("%s: %v", rq.Name, err)
		}
		out = append(out, pendingFile{ReqID: rq.ID, Header: fhs[0], Ext: ext})
	}
	return out, nil
}

func (h *Handler) saveFile(p pendingFile) (string, error) {
	src, err := p.Header.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	full := filepath.Join(h.Cfg.UploadDir, hex.EncodeToString(b)+p.Ext)
	if abs, err := filepath.Abs(full); err == nil {
		full = abs
	}
	dst, err := os.Create(full)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}
	return full, nil
}

func (h *Handler) insertDocs(ctx context.Context, tx pgx.Tx, ticketID int, files []pendingFile) error {
	for _, p := range files {
		path, err := h.saveFile(p)
		if err != nil {
			return err
		}
		mime := "image/jpeg"
		if p.Ext == ".pdf" {
			mime = "application/pdf"
		}
		var version int
		if err := tx.QueryRow(ctx,
			`SELECT COALESCE(MAX(version),0)+1 FROM ticket_documents WHERE ticket_id=$1 AND requirement_id=$2`,
			ticketID, p.ReqID).Scan(&version); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`UPDATE ticket_documents SET is_current=false WHERE ticket_id=$1 AND requirement_id=$2`,
			ticketID, p.ReqID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO ticket_documents (ticket_id, requirement_id, file_name, file_path, mime, size_bytes, version)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			ticketID, p.ReqID, filepath.Base(p.Header.Filename), path, mime, p.Header.Size, version); err != nil {
			return err
		}
	}
	return nil
}

func parseUpload(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, 40<<20)
	return r.ParseMultipartForm(8 << 20)
}

// ---------- Endpoint mahasiswa ----------

// US-03: kirim pengajuan. Jenis surat tertentu diteruskan dulu ke Dosen PA (US-09).
func (h *Handler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	if err := parseUpload(w, r); err != nil {
		writeErr(w, http.StatusBadRequest, "Form tidak valid atau ukuran total terlalu besar")
		return
	}
	ctx := r.Context()
	letterID, _ := strconv.Atoi(r.FormValue("letter_type_id"))
	purpose := strings.TrimSpace(r.FormValue("purpose"))
	if r.FormValue("confirmed") != "true" {
		writeErr(w, http.StatusBadRequest, "Lembar konfirmasi validasi data harus disetujui")
		return
	}
	if purpose == "" {
		writeErr(w, http.StatusBadRequest, "Keperluan surat wajib diisi")
		return
	}

	var requires bool
	if err := h.DB.QueryRow(ctx,
		`SELECT requires_recommendation FROM letter_types WHERE id=$1 AND active`, letterID).
		Scan(&requires); err != nil {
		writeErr(w, http.StatusNotFound, "Jenis surat tidak ditemukan")
		return
	}
	if requires {
		var adv *int
		_ = h.DB.QueryRow(ctx, `SELECT advisor_id FROM users WHERE id=$1`, c.UserID).Scan(&adv)
		if adv == nil {
			writeErr(w, http.StatusBadRequest, "Akun Anda belum memiliki Dosen PA. Hubungi staf administrasi akademik.")
			return
		}
	}
	reqs, err := h.requirements(ctx, letterID)
	if err != nil || len(reqs) == 0 {
		writeErr(w, http.StatusNotFound, "Persyaratan surat belum diatur")
		return
	}
	files, err := h.collectFiles(r, reqs, true)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memproses pengajuan")
		return
	}
	defer tx.Rollback(ctx)

	status := "submitted"
	if requires {
		status = "awaiting_recommendation"
	}
	var seq, id int
	if err := tx.QueryRow(ctx, `SELECT nextval('ticket_seq')`).Scan(&seq); err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal membuat nomor tiket")
		return
	}
	no := fmt.Sprintf("SRT-%s-%04d", time.Now().Format("20060102"), seq)
	if err := tx.QueryRow(ctx, `
		INSERT INTO tickets (ticket_no, student_id, letter_type_id, purpose, status, verifier_unit)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		no, c.UserID, letterID, purpose, status, statusUnit[status]).Scan(&id); err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal menyimpan pengajuan")
		return
	}
	if err := h.insertDocs(ctx, tx, id, files); err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal menyimpan berkas")
		return
	}
	if err := addHistory(ctx, tx, id, "submitted", "Pengajuan diterima sistem"); err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal mencatat riwayat")
		return
	}
	if requires {
		if err := addHistory(ctx, tx, id, "awaiting_recommendation", "Menunggu rekomendasi Dosen PA"); err != nil {
			writeErr(w, http.StatusInternalServerError, "Gagal mencatat riwayat")
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal menyimpan pengajuan")
		return
	}
	h.audit(ctx, c.UserID, "mengajukan surat "+no)
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "ticket_no": no, "status": status})
}

func (h *Handler) MyTickets(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	items, err := h.listTickets(r.Context(), "WHERE t.student_id=$1", "t.created_at DESC", c.UserID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memuat pengajuan")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// US-04: detail + pelacakan
func (h *Handler) GetTicket(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeErr(w, http.StatusNotFound, "Pengajuan tidak ditemukan")
		return
	}
	t, err := scanTicket(h.DB.QueryRow(r.Context(), ticketSelect+"WHERE t.id=$1", id))
	if err != nil || !h.canView(r.Context(), c, t) {
		writeErr(w, http.StatusNotFound, "Pengajuan tidak ditemukan")
		return
	}
	reqs, _ := h.requirements(r.Context(), t.LetterTypeID)
	writeJSON(w, http.StatusOK, TicketDetail{
		Ticket:       t,
		History:      h.history(r.Context(), id),
		Documents:    h.documents(r.Context(), id),
		Requirements: reqs,
	})
}

// US-05: unggah ulang berkas pada tiket yang sama
func (h *Handler) Resubmit(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeErr(w, http.StatusNotFound, "Pengajuan tidak ditemukan")
		return
	}
	ctx := r.Context()

	var studentID, letterID int
	var status, ticketNo string
	err = h.DB.QueryRow(ctx, `SELECT student_id, letter_type_id, status, ticket_no FROM tickets WHERE id=$1`, id).
		Scan(&studentID, &letterID, &status, &ticketNo)
	if err != nil || studentID != c.UserID {
		writeErr(w, http.StatusNotFound, "Pengajuan tidak ditemukan")
		return
	}
	if status != "revision" {
		writeErr(w, http.StatusConflict, "Pengajuan ini tidak sedang meminta revisi")
		return
	}
	if err := parseUpload(w, r); err != nil {
		writeErr(w, http.StatusBadRequest, "Form tidak valid atau ukuran total terlalu besar")
		return
	}
	reqs, _ := h.requirements(ctx, letterID)
	files, err := h.collectFiles(r, reqs, false)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(files) == 0 {
		writeErr(w, http.StatusBadRequest, "Unggah minimal satu dokumen pengganti")
		return
	}

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memproses revisi")
		return
	}
	defer tx.Rollback(ctx)

	if err := h.insertDocs(ctx, tx, id, files); err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal menyimpan berkas")
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE tickets SET status='reviewing', note='', verifier_unit=$1, updated_at=now() WHERE id=$2`,
		statusUnit["reviewing"], id); err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memperbarui status")
		return
	}
	if err := addHistory(ctx, tx, id, "reviewing", "Mahasiswa mengunggah ulang berkas revisi"); err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal mencatat riwayat")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal menyimpan revisi")
		return
	}
	h.audit(ctx, c.UserID, "mengunggah ulang berkas revisi "+ticketNo)
	writeJSON(w, http.StatusOK, map[string]string{"status": "reviewing"})
}

// Unduh/preview dokumen pendukung (pemilik, staf, dosen bimbingan)
func (h *Handler) DocumentFile(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeErr(w, http.StatusNotFound, "Dokumen tidak ditemukan")
		return
	}
	var path, mime, name string
	var studentID, ticketID int
	err = h.DB.QueryRow(r.Context(), `
		SELECT d.file_path, d.mime, d.file_name, t.student_id, t.id
		FROM ticket_documents d JOIN tickets t ON t.id = d.ticket_id WHERE d.id=$1`, id).
		Scan(&path, &mime, &name, &studentID, &ticketID)
	if err != nil ||
		(c.Role == "mahasiswa" && studentID != c.UserID) ||
		(c.Role == "dosen" && !h.advises(r.Context(), c.UserID, ticketID)) {
		writeErr(w, http.StatusNotFound, "Dokumen tidak ditemukan")
		return
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		alt := filepath.Join(h.Cfg.UploadDir, filepath.Base(path))
		if _, err2 := os.Stat(alt); err2 == nil {
			path = alt
		}
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", name))
	http.ServeFile(w, r, path)
}

// ---------- Endpoint staf ----------

// US-10: antrean terpusat berdasarkan waktu pengajuan (yang masih menunggu dosen tidak ditampilkan)
func (h *Handler) StaffTickets(w http.ResponseWriter, r *http.Request) {
	where := "WHERE t.status <> 'awaiting_recommendation'"
	var args []any
	if s := r.URL.Query().Get("status"); s != "" {
		where += " AND t.status=$1"
		args = append(args, s)
	}
	items, err := h.listTickets(r.Context(), where,
		"CASE WHEN t.status IN ('completed','rejected') THEN 1 ELSE 0 END, t.created_at", args...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memuat antrean")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// staf membuka berkas -> status "Pemeriksaan Berkas"
func (h *Handler) StartReview(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeErr(w, http.StatusNotFound, "Pengajuan tidak ditemukan")
		return
	}
	ctx := r.Context()
	res, err := h.DB.Exec(ctx,
		`UPDATE tickets SET status='reviewing', verifier_unit=$1, updated_at=now() WHERE id=$2 AND status='submitted'`,
		statusUnit["reviewing"], id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memperbarui status")
		return
	}
	if res.RowsAffected() == 1 {
		_, _ = h.DB.Exec(ctx,
			`INSERT INTO ticket_history (ticket_id, status, unit, note) VALUES ($1,'reviewing',$2,'Berkas sedang diperiksa staf')`,
			id, statusUnit["reviewing"])
		var no string
		_ = h.DB.QueryRow(ctx, `SELECT ticket_no FROM tickets WHERE id=$1`, id).Scan(&no)
		h.audit(ctx, c.UserID, "memeriksa berkas "+no)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// US-05, US-10, US-11: keputusan staf. "approving" = berkas valid -> nomor surat + draf PDF.
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeErr(w, http.StatusNotFound, "Pengajuan tidak ditemukan")
		return
	}
	var in struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "Data tidak valid")
		return
	}
	in.Note = strings.TrimSpace(in.Note)
	allowed := map[string]bool{"revision": true, "approving": true, "rejected": true}
	if !allowed[in.Status] {
		writeErr(w, http.StatusBadRequest, "Status tidak valid")
		return
	}
	if (in.Status == "revision" || in.Status == "rejected") && in.Note == "" {
		writeErr(w, http.StatusBadRequest, "Catatan wajib diisi untuk revisi atau penolakan")
		return
	}

	ctx := r.Context()
	var studentID int
	var ticketNo, cur string
	if err := h.DB.QueryRow(ctx, `SELECT student_id, ticket_no, status FROM tickets WHERE id=$1`, id).
		Scan(&studentID, &ticketNo, &cur); err != nil {
		writeErr(w, http.StatusNotFound, "Pengajuan tidak ditemukan")
		return
	}
	if cur != "submitted" && cur != "reviewing" {
		writeErr(w, http.StatusConflict, "Pengajuan tidak berada pada tahap pemeriksaan berkas")
		return
	}

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memproses")
		return
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`UPDATE tickets SET status=$1, note=$2, verifier_unit=$3, updated_at=now() WHERE id=$4`,
		in.Status, in.Note, statusUnit[in.Status], id); err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memperbarui status")
		return
	}

	histNote := in.Note
	letterNo := ""
	if in.Status == "approving" {
		letterNo, err = h.createDraft(ctx, tx, id)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "Gagal membuat nomor dan draf surat")
			return
		}
		histNote = strings.TrimSpace("Berkas lengkap. Nomor surat: " + letterNo + ". " + in.Note)
	}
	if err := addHistory(ctx, tx, id, in.Status, histNote); err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal mencatat riwayat")
		return
	}
	if in.Status == "revision" {
		body := fmt.Sprintf("Pengajuan %s memerlukan revisi: %s", ticketNo, in.Note)
		if _, err := tx.Exec(ctx,
			`INSERT INTO messages (student_id, sender_id, sender_role, body) VALUES ($1,$2,'staf',$3)`,
			studentID, c.UserID, body); err != nil {
			writeErr(w, http.StatusInternalServerError, "Gagal mengirim notifikasi")
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal menyimpan")
		return
	}

	switch in.Status {
	case "revision":
		h.audit(ctx, c.UserID, "meminta revisi "+ticketNo)
	case "rejected":
		h.audit(ctx, c.UserID, "menolak surat "+ticketNo)
	case "approving":
		h.audit(ctx, c.UserID, "menyetujui surat "+ticketNo)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": in.Status, "letter_no": letterNo})
}
