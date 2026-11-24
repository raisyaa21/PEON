package handlers

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"rsi-surat/internal/middleware"
)

type AcademicRecord struct {
	Semester int     `json:"semester"`
	IPS      float64 `json:"ips"`
	SKS      int     `json:"sks"`
}

type DosenDetail struct {
	TicketDetail
	Program  string           `json:"program"`
	Semester int              `json:"semester"`
	Records  []AcademicRecord `json:"records"`
	IPK      float64          `json:"ipk"`
}

// US-09: daftar permohonan bimbingan
func (h *Handler) DosenList(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	items, err := h.listTickets(r.Context(),
		"WHERE lt.requires_recommendation AND u.advisor_id=$1", "t.created_at DESC", c.UserID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memuat permohonan")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) DosenDetail(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	ctx := r.Context()
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeErr(w, http.StatusNotFound, "Permohonan tidak ditemukan")
		return
	}
	t, err := scanTicket(h.DB.QueryRow(ctx, ticketSelect+"WHERE t.id=$1", id))
	if err != nil || !h.advises(ctx, c.UserID, id) {
		writeErr(w, http.StatusNotFound, "Permohonan tidak ditemukan")
		return
	}
	reqs, _ := h.requirements(ctx, t.LetterTypeID)
	d := DosenDetail{
		TicketDetail: TicketDetail{
			Ticket:       t,
			History:      h.history(ctx, id),
			Documents:    h.documents(ctx, id),
			Requirements: reqs,
		},
		Records: []AcademicRecord{},
	}
	_ = h.DB.QueryRow(ctx, `SELECT study_program, semester FROM users WHERE id=$1`, t.StudentID).
		Scan(&d.Program, &d.Semester)

	rows, err := h.DB.Query(ctx,
		`SELECT semester, ips, sks FROM academic_records WHERE student_id=$1 ORDER BY semester`, t.StudentID)
	if err == nil {
		defer rows.Close()
		var sum, sks float64
		for rows.Next() {
			var a AcademicRecord
			if rows.Scan(&a.Semester, &a.IPS, &a.SKS) == nil {
				d.Records = append(d.Records, a)
				sum += a.IPS * float64(a.SKS)
				sks += float64(a.SKS)
			}
		}
		if sks > 0 {
			d.IPK = math.Round(sum/sks*100) / 100
		}
	}
	writeJSON(w, http.StatusOK, d)
}

// US-09: rekomendasikan / tolak -> status tiket otomatis berubah dan diteruskan ke staf
func (h *Handler) DosenDecision(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	ctx := r.Context()
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeErr(w, http.StatusNotFound, "Permohonan tidak ditemukan")
		return
	}
	var in struct {
		Decision string `json:"decision"`
		Note     string `json:"note"`
	}
	if err := decode(r, &in); err != nil || (in.Decision != "recommended" && in.Decision != "rejected") {
		writeErr(w, http.StatusBadRequest, "Keputusan tidak valid")
		return
	}
	in.Note = strings.TrimSpace(in.Note)
	if in.Decision == "rejected" && in.Note == "" {
		writeErr(w, http.StatusBadRequest, "Catatan wajib diisi untuk penolakan")
		return
	}
	if !h.advises(ctx, c.UserID, id) {
		writeErr(w, http.StatusNotFound, "Permohonan tidak ditemukan")
		return
	}
	var cur, ticketNo string
	if err := h.DB.QueryRow(ctx, `SELECT status, ticket_no FROM tickets WHERE id=$1`, id).Scan(&cur, &ticketNo); err != nil {
		writeErr(w, http.StatusNotFound, "Permohonan tidak ditemukan")
		return
	}
	if cur != "awaiting_recommendation" {
		writeErr(w, http.StatusConflict, "Permohonan ini sudah diproses")
		return
	}

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memproses")
		return
	}
	defer tx.Rollback(ctx)

	const unit = "Dosen Pembimbing Akademik"
	var action string
	if in.Decision == "recommended" {
		note := "Direkomendasikan Dosen PA"
		if in.Note != "" {
			note += ": " + in.Note
		}
		if _, err = tx.Exec(ctx, `UPDATE tickets SET status='submitted', advisor_status='recommended', advisor_note=$1,
			verifier_unit=$2, updated_at=now() WHERE id=$3`, in.Note, unit, id); err == nil {
			_, err = tx.Exec(ctx, `INSERT INTO ticket_history (ticket_id, status, unit, note) VALUES ($1,'submitted',$2,$3)`,
				id, unit, note)
		}
		action = "menerbitkan rekomendasi " + ticketNo
	} else {
		if _, err = tx.Exec(ctx, `UPDATE tickets SET status='rejected', advisor_status='rejected', advisor_note=$1,
			note=$1, verifier_unit=$2, updated_at=now() WHERE id=$3`, in.Note, unit, id); err == nil {
			_, err = tx.Exec(ctx, `INSERT INTO ticket_history (ticket_id, status, unit, note) VALUES ($1,'rejected',$2,$3)`,
				id, unit, "Ditolak Dosen PA: "+in.Note)
		}
		action = "menolak rekomendasi " + ticketNo
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal menyimpan keputusan")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal menyimpan keputusan")
		return
	}
	h.audit(ctx, c.UserID, action)
	writeJSON(w, http.StatusOK, map[string]string{"result": fmt.Sprint(in.Decision)})
}
