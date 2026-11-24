package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	qrcode "github.com/skip2/go-qrcode"

	"rsi-surat/internal/middleware"
)

var monthID = []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli",
	"Agustus", "September", "Oktober", "November", "Desember"}

var romanMonth = []string{"", "I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "XI", "XII"}

const defaultTemplate = `Yang bertanda tangan di bawah ini menerangkan bahwa:

Nama : {{nama}}
NIM : {{nim}}
Program Studi : {{prodi}}

mengajukan {{jenis}} untuk keperluan: {{keperluan}}.

Demikian surat ini dibuat untuk dipergunakan sebagaimana mestinya.`

func dateID(t time.Time) string {
	return fmt.Sprintf("%d %s %d", t.Day(), monthID[t.Month()], t.Year())
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// US-11: pola {NO}/{KODE}/{JUR}/{BLN}/{THN}
func formatNumber(f string, seq int, code, dept string, t time.Time) string {
	return strings.NewReplacer(
		"{NO}", fmt.Sprintf("%03d", seq),
		"{KODE}", code,
		"{JUR}", dept,
		"{BLN}", romanMonth[t.Month()],
		"{THN}", strconv.Itoa(t.Year()),
	).Replace(f)
}

type letterSource struct {
	TicketID     int
	TicketNo     string
	StudentName  string
	StudentNIM   string
	Program      string
	Semester     int
	Purpose      string
	LetterTypeID int
	Code         string
	TypeName     string
	NumberFormat string
	Template     string
}

func loadSource(ctx context.Context, q dbq, ticketID int) (letterSource, error) {
	var s letterSource
	err := q.QueryRow(ctx, `
		SELECT t.id, t.ticket_no, u.name, u.identifier, u.study_program, u.semester, t.purpose,
		       lt.id, lt.code, lt.name, lt.number_format, lt.template_body
		FROM tickets t
		JOIN users u ON u.id = t.student_id
		JOIN letter_types lt ON lt.id = t.letter_type_id
		WHERE t.id=$1`, ticketID).
		Scan(&s.TicketID, &s.TicketNo, &s.StudentName, &s.StudentNIM, &s.Program, &s.Semester, &s.Purpose,
			&s.LetterTypeID, &s.Code, &s.TypeName, &s.NumberFormat, &s.Template)
	return s, err
}

func loadSettings(ctx context.Context, q dbq) map[string]string {
	m := map[string]string{}
	rows, err := q.Query(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if rows.Scan(&k, &v) == nil {
			m[k] = v
		}
	}
	return m
}

func buildLetter(src letterSource, set map[string]string, letterNo string, at time.Time) letterData {
	tmpl := strings.TrimSpace(src.Template)
	if tmpl == "" {
		tmpl = defaultTemplate
	}
	prodi := src.Program
	if prodi == "" {
		prodi = "Sistem Informasi"
	}
	body := strings.NewReplacer(
		"{{nama}}", src.StudentName,
		"{{nim}}", src.StudentNIM,
		"{{prodi}}", prodi,
		"{{semester}}", strconv.Itoa(src.Semester),
		"{{keperluan}}", src.Purpose,
		"{{nomor}}", letterNo,
		"{{tanggal}}", dateID(at),
		"{{jenis}}", src.TypeName,
		"{{jabatan}}", set["signer_title"],
	).Replace(tmpl)
	city := set["city"]
	if city == "" {
		city = "Malang"
	}
	return letterData{
		Kop:         []string{set["kop_1"], set["kop_2"], set["kop_3"]},
		Address:     set["address"],
		Title:       src.TypeName,
		LetterNo:    letterNo,
		Body:        body,
		City:        city,
		Date:        at,
		SignerName:  set["signer_name"],
		SignerTitle: set["signer_title"],
	}
}

func (h *Handler) writeLetterFile(name string, data []byte) (string, error) {
	dir := filepath.Join(h.Cfg.UploadDir, "letters")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	p := filepath.Join(dir, name)
	return p, os.WriteFile(p, data, 0o644)
}

// US-11: nomor surat unik (counter per jenis surat per tahun) + draf PDF berkop. Dipanggil di dalam transaksi.
func (h *Handler) createDraft(ctx context.Context, tx pgx.Tx, ticketID int) (string, error) {
	src, err := loadSource(ctx, tx, ticketID)
	if err != nil {
		return "", err
	}
	now := time.Now()
	var seq int
	if err := tx.QueryRow(ctx, `
		INSERT INTO letter_counters (letter_type_id, year, last_no) VALUES ($1,$2,1)
		ON CONFLICT (letter_type_id, year) DO UPDATE SET last_no = letter_counters.last_no + 1
		RETURNING last_no`, src.LetterTypeID, now.Year()).Scan(&seq); err != nil {
		return "", err
	}
	no := formatNumber(src.NumberFormat, seq, src.Code, h.Cfg.DeptCode, now)
	code := randHex(12)

	set := loadSettings(ctx, tx)
	pdfBytes, err := renderLetterPDF(buildLetter(src, set, no, now))
	if err != nil {
		return "", err
	}
	path, err := h.writeLetterFile("draft-"+code+".pdf", pdfBytes)
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO letters (ticket_id, letter_no, verify_code, draft_path) VALUES ($1,$2,$3,$4)
		 ON CONFLICT (ticket_id) DO UPDATE SET letter_no=$2, verify_code=$3, draft_path=$4`,
		ticketID, no, code, path); err != nil {
		return "", err
	}
	return no, nil
}

// US-12: staf menerbitkan surat final (TTE + QR) dan status menjadi "completed"
func (h *Handler) IssueLetter(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	ctx := r.Context()
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeErr(w, http.StatusNotFound, "Pengajuan tidak ditemukan")
		return
	}

	var status, ticketNo string
	var studentID int
	if err := h.DB.QueryRow(ctx, `SELECT status, student_id, ticket_no FROM tickets WHERE id=$1`, id).
		Scan(&status, &studentID, &ticketNo); err != nil {
		writeErr(w, http.StatusNotFound, "Pengajuan tidak ditemukan")
		return
	}
	if status != "approving" {
		writeErr(w, http.StatusConflict, "Surat hanya dapat diterbitkan pada tahap Pengesahan Surat")
		return
	}
	src, err := loadSource(ctx, h.DB, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memuat data surat")
		return
	}
	var letterNo, code string
	if err := h.DB.QueryRow(ctx, `SELECT letter_no, verify_code FROM letters WHERE ticket_id=$1`, id).
		Scan(&letterNo, &code); err != nil {
		writeErr(w, http.StatusConflict, "Draf surat belum dibuat")
		return
	}

	set := loadSettings(ctx, h.DB)
	now := time.Now()
	verifyURL := h.Cfg.FrontendOrigin + "/verifikasi/" + code
	qr, err := qrcode.Encode(verifyURL, qrcode.Medium, 256)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal membuat QR code")
		return
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%s|%d",
		letterNo, src.StudentNIM, src.TypeName, src.Purpose, now.Unix())))
	hash := hex.EncodeToString(sum[:])

	d := buildLetter(src, set, letterNo, now)
	d.Final, d.QRPNG, d.VerifyURL, d.DocHash = true, qr, verifyURL, hash
	pdfBytes, err := renderLetterPDF(d)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal membuat PDF final")
		return
	}
	path, err := h.writeLetterFile("final-"+code+".pdf", pdfBytes)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal menyimpan PDF final")
		return
	}

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memproses")
		return
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`UPDATE letters SET final_path=$1, signed_by=$2, doc_hash=$3, issued_at=$4 WHERE ticket_id=$5`,
		path, set["signer_name"], hash, now, id); err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memperbarui surat")
		return
	}
	if _, err := tx.Exec(ctx,
		`UPDATE tickets SET status='completed', verifier_unit=$1, updated_at=now() WHERE id=$2`,
		statusUnit["completed"], id); err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memperbarui status")
		return
	}
	if err := addHistory(ctx, tx, id, "completed", "Surat diterbitkan dengan TTE. Nomor: "+letterNo); err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal mencatat riwayat")
		return
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO messages (student_id, sender_id, sender_role, body) VALUES ($1,$2,'staf',$3)`,
		studentID, c.UserID, fmt.Sprintf("Surat %s (%s) telah terbit. Unduh melalui menu Riwayat.", src.TypeName, letterNo)); err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal mengirim notifikasi")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal menyimpan")
		return
	}
	h.audit(ctx, c.UserID, "menerbitkan surat "+letterNo)
	writeJSON(w, http.StatusOK, map[string]string{"status": "completed", "letter_no": letterNo})
}

// GET /api/tickets/{id}/letter?kind=final|draft
func (h *Handler) LetterFile(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeErr(w, http.StatusNotFound, "Surat tidak ditemukan")
		return
	}
	kind := r.URL.Query().Get("kind")
	if kind != "draft" {
		kind = "final"
	}
	var studentID int
	var draft, final, ticketNo string
	if err := h.DB.QueryRow(r.Context(), `
		SELECT t.student_id, l.draft_path, l.final_path, t.ticket_no
		FROM letters l JOIN tickets t ON t.id = l.ticket_id WHERE l.ticket_id=$1`, id).
		Scan(&studentID, &draft, &final, &ticketNo); err != nil {
		writeErr(w, http.StatusNotFound, "Surat tidak ditemukan")
		return
	}
	if c.Role == "mahasiswa" && (studentID != c.UserID || kind != "final") {
		writeErr(w, http.StatusNotFound, "Surat tidak ditemukan")
		return
	}
	if c.Role == "dosen" && !h.advises(r.Context(), c.UserID, id) {
		writeErr(w, http.StatusNotFound, "Surat tidak ditemukan")
		return
	}
	path := final
	if kind == "draft" {
		path = draft
	}
	if path == "" {
		writeErr(w, http.StatusNotFound, "Surat belum tersedia")
		return
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		alt := filepath.Join(h.Cfg.UploadDir, "letters", filepath.Base(path))
		if _, err2 := os.Stat(alt); err2 == nil {
			path = alt
		}
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", "Surat-"+ticketNo+".pdf"))
	http.ServeFile(w, r, path)
}

// US-08: arsip surat mahasiswa
func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	rows, err := h.DB.Query(r.Context(), `
		SELECT t.id, t.ticket_no, lt.name, l.letter_no, l.issued_at, l.final_path
		FROM letters l
		JOIN tickets t ON t.id = l.ticket_id
		JOIN letter_types lt ON lt.id = t.letter_type_id
		WHERE t.student_id=$1 AND l.final_path <> ''
		ORDER BY l.issued_at DESC`, c.UserID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memuat arsip")
		return
	}
	defer rows.Close()
	type item struct {
		TicketID   int       `json:"ticket_id"`
		TicketNo   string    `json:"ticket_no"`
		LetterType string    `json:"letter_type"`
		LetterNo   string    `json:"letter_no"`
		IssuedAt   time.Time `json:"issued_at"`
		SizeKB     int64     `json:"size_kb"`
	}
	out := []item{}
	for rows.Next() {
		var it item
		var path string
		if err := rows.Scan(&it.TicketID, &it.TicketNo, &it.LetterType, &it.LetterNo, &it.IssuedAt, &path); err != nil {
			continue
		}
		if st, err := os.Stat(path); err == nil {
			it.SizeKB = st.Size() / 1024
		}
		out = append(out, it)
	}
	writeJSON(w, http.StatusOK, out)
}

// publik: verifikasi keaslian surat dari QR
func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	var no, typ, name, nim, signer, hash string
	var issued time.Time
	err := h.DB.QueryRow(r.Context(), `
		SELECT l.letter_no, lt.name, u.name, u.identifier, l.signed_by, l.doc_hash, l.issued_at
		FROM letters l
		JOIN tickets t ON t.id = l.ticket_id
		JOIN letter_types lt ON lt.id = t.letter_type_id
		JOIN users u ON u.id = t.student_id
		WHERE l.verify_code=$1 AND l.final_path <> ''`, code).
		Scan(&no, &typ, &name, &nim, &signer, &hash, &issued)
	if err != nil {
		writeErr(w, http.StatusNotFound, "Dokumen tidak ditemukan atau belum diterbitkan")
		return
	}
	if len(nim) > 7 {
		nim = nim[:4] + strings.Repeat("•", len(nim)-7) + nim[len(nim)-3:]
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"valid": true, "letter_no": no, "letter_type": typ, "student_name": name,
		"student_nim": nim, "signed_by": signer, "issued_at": issued, "hash": hash[:16],
	})
}
