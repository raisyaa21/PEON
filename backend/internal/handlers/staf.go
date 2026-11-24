package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"rsi-surat/internal/middleware"
)

// ================= US-13: layanan & template =================

type AdminLetter struct {
	ID                     int           `json:"id"`
	Code                   string        `json:"code"`
	Name                   string        `json:"name"`
	Description            string        `json:"description"`
	Active                 bool          `json:"active"`
	RequiresRecommendation bool          `json:"requires_recommendation"`
	NumberFormat           string        `json:"number_format"`
	TemplateBody           string        `json:"template_body"`
	Requirements           []Requirement `json:"requirements"`
}

func (h *Handler) queryAdminLetters(ctx context.Context, id int) ([]AdminLetter, error) {
	rows, err := h.DB.Query(ctx, `
		SELECT lt.id, lt.code, lt.name, lt.description, lt.active, lt.requires_recommendation,
		       lt.number_format, lt.template_body,
		       COALESCE(json_agg(json_build_object('id', lr.id, 'name', lr.name) ORDER BY lr.id)
		                FILTER (WHERE lr.id IS NOT NULL), '[]'::json)
		FROM letter_types lt
		LEFT JOIN letter_requirements lr ON lr.letter_type_id = lt.id
		WHERE ($1::int = 0 OR lt.id = $1::int)
		GROUP BY lt.id ORDER BY lt.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AdminLetter{}
	for rows.Next() {
		var l AdminLetter
		if err := rows.Scan(&l.ID, &l.Code, &l.Name, &l.Description, &l.Active, &l.RequiresRecommendation,
			&l.NumberFormat, &l.TemplateBody, &l.Requirements); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (h *Handler) AdminListLetters(w http.ResponseWriter, r *http.Request) {
	items, err := h.queryAdminLetters(r.Context(), 0)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memuat layanan")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) AdminGetLetter(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeErr(w, http.StatusNotFound, "Layanan tidak ditemukan")
		return
	}
	items, err := h.queryAdminLetters(r.Context(), id)
	if err != nil || len(items) == 0 {
		writeErr(w, http.StatusNotFound, "Layanan tidak ditemukan")
		return
	}
	writeJSON(w, http.StatusOK, items[0])
}

type letterIn struct {
	Code                   string        `json:"code"`
	Name                   string        `json:"name"`
	Description            string        `json:"description"`
	Active                 *bool         `json:"active"`
	RequiresRecommendation bool          `json:"requires_recommendation"`
	NumberFormat           string        `json:"number_format"`
	TemplateBody           string        `json:"template_body"`
	Requirements           []Requirement `json:"requirements"`
}

func validateLetter(in *letterIn) string {
	in.Code = strings.ToUpper(strings.TrimSpace(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	in.NumberFormat = strings.TrimSpace(in.NumberFormat)
	if in.Code == "" || in.Name == "" {
		return "Kode dan nama layanan wajib diisi"
	}
	if !strings.Contains(in.NumberFormat, "{NO}") {
		return "Format penomoran harus memuat {NO}"
	}
	if strings.TrimSpace(in.TemplateBody) == "" {
		return "Template surat wajib diisi"
	}
	clean := []Requirement{}
	for _, r := range in.Requirements {
		if n := strings.TrimSpace(r.Name); n != "" {
			clean = append(clean, Requirement{ID: r.ID, Name: n})
		}
	}
	if len(clean) == 0 {
		return "Minimal satu dokumen persyaratan harus diisi"
	}
	in.Requirements = clean
	return ""
}

func (h *Handler) AdminCreateLetter(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	ctx := r.Context()
	var in letterIn
	if err := decode(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "Data tidak valid")
		return
	}
	if msg := validateLetter(&in); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memproses")
		return
	}
	defer tx.Rollback(ctx)

	var id int
	if err := tx.QueryRow(ctx, `
		INSERT INTO letter_types (code, name, description, requires_recommendation, number_format, template_body)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		in.Code, in.Name, in.Description, in.RequiresRecommendation, in.NumberFormat, in.TemplateBody).Scan(&id); err != nil {
		if strings.Contains(err.Error(), "23505") {
			writeErr(w, http.StatusConflict, "Kode layanan sudah dipakai")
			return
		}
		writeErr(w, http.StatusInternalServerError, "Gagal menyimpan layanan")
		return
	}
	for _, rq := range in.Requirements {
		if _, err := tx.Exec(ctx, `INSERT INTO letter_requirements (letter_type_id, name) VALUES ($1,$2)`, id, rq.Name); err != nil {
			writeErr(w, http.StatusInternalServerError, "Gagal menyimpan persyaratan")
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal menyimpan")
		return
	}
	h.audit(ctx, c.UserID, "menambahkan layanan surat "+in.Name+" ("+in.Code+")")
	writeJSON(w, http.StatusCreated, map[string]int{"id": id})
}

func (h *Handler) AdminUpdateLetter(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	ctx := r.Context()
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeErr(w, http.StatusNotFound, "Layanan tidak ditemukan")
		return
	}
	var in letterIn
	if err := decode(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "Data tidak valid")
		return
	}
	if msg := validateLetter(&in); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memproses")
		return
	}
	defer tx.Rollback(ctx)

	res, err := tx.Exec(ctx, `
		UPDATE letter_types SET code=$1, name=$2, description=$3, requires_recommendation=$4,
		       number_format=$5, template_body=$6, active=$7 WHERE id=$8`,
		in.Code, in.Name, in.Description, in.RequiresRecommendation, in.NumberFormat, in.TemplateBody, active, id)
	if err != nil {
		if strings.Contains(err.Error(), "23505") {
			writeErr(w, http.StatusConflict, "Kode layanan sudah dipakai")
			return
		}
		writeErr(w, http.StatusInternalServerError, "Gagal menyimpan layanan")
		return
	}
	if res.RowsAffected() == 0 {
		writeErr(w, http.StatusNotFound, "Layanan tidak ditemukan")
		return
	}

	keep := make([]int32, 0, len(in.Requirements))
	for _, rq := range in.Requirements {
		if rq.ID > 0 {
			keep = append(keep, int32(rq.ID))
			if _, err := tx.Exec(ctx, `UPDATE letter_requirements SET name=$1 WHERE id=$2 AND letter_type_id=$3`,
				rq.Name, rq.ID, id); err != nil {
				writeErr(w, http.StatusInternalServerError, "Gagal memperbarui persyaratan")
				return
			}
		}
	}
	// hapus persyaratan yang dibuang, kecuali yang sudah dipakai pengajuan
	if _, err := tx.Exec(ctx, `
		DELETE FROM letter_requirements lr
		WHERE lr.letter_type_id=$1 AND lr.id <> ALL($2::int[])
		  AND NOT EXISTS (SELECT 1 FROM ticket_documents d WHERE d.requirement_id = lr.id)`, id, keep); err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memperbarui persyaratan")
		return
	}
	for _, rq := range in.Requirements {
		if rq.ID == 0 {
			if _, err := tx.Exec(ctx, `INSERT INTO letter_requirements (letter_type_id, name) VALUES ($1,$2)`, id, rq.Name); err != nil {
				writeErr(w, http.StatusInternalServerError, "Gagal menambah persyaratan")
				return
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal menyimpan")
		return
	}
	h.audit(ctx, c.UserID, "memperbarui layanan surat "+in.Name+" ("+in.Code+")")
	writeJSON(w, http.StatusOK, map[string]int{"id": id})
}

// ================= pengaturan kop surat & penandatangan =================

var settingKeys = []string{"kop_1", "kop_2", "kop_3", "address", "city", "signer_name", "signer_title"}

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, loadSettings(r.Context(), h.DB))
}

func (h *Handler) PutSettings(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	var in map[string]string
	if err := decode(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "Data tidak valid")
		return
	}
	for _, k := range settingKeys {
		if v, ok := in[k]; ok {
			if _, err := h.DB.Exec(r.Context(), `
				INSERT INTO settings (key, value) VALUES ($1,$2)
				ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, k, strings.TrimSpace(v)); err != nil {
				writeErr(w, http.StatusInternalServerError, "Gagal menyimpan pengaturan")
				return
			}
		}
	}
	h.audit(r.Context(), c.UserID, "memperbarui kop surat dan penandatangan")
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ================= US-14: hak akses & audit log =================

type AdminUser struct {
	ID         int    `json:"id"`
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	Position   string `json:"position"`
	Active     bool   `json:"active"`
}

func (h *Handler) AdminUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `
		SELECT id, identifier, name, role, position, active FROM users
		WHERE ($1::text = '' OR name ILIKE '%' || $1::text || '%' OR identifier ILIKE '%' || $1::text || '%')
		ORDER BY CASE role WHEN 'staf' THEN 0 WHEN 'dosen' THEN 1 ELSE 2 END, name`,
		strings.TrimSpace(r.URL.Query().Get("q")))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memuat pengguna")
		return
	}
	defer rows.Close()
	out := []AdminUser{}
	for rows.Next() {
		var u AdminUser
		if rows.Scan(&u.ID, &u.Identifier, &u.Name, &u.Role, &u.Position, &u.Active) == nil {
			out = append(out, u)
		}
	}
	writeJSON(w, http.StatusOK, out)
}

var accessMap = map[string]struct{ Role, Position string }{
	"dosen_pa":      {"dosen", "Dosen Pembimbing Akademik (Dosen PA)"},
	"dosen_skripsi": {"dosen", "Dosen Pembimbing Skripsi / Tugas Akhir (Dospem)"},
	"dosen_dpl":     {"dosen", "Dosen Pembimbing Lapangan (DPL)"},
	"staf":          {"staf", "Staf Administrasi & Operasional"},
}

func (h *Handler) AdminSetAccess(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	ctx := r.Context()
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeErr(w, http.StatusNotFound, "Pengguna tidak ditemukan")
		return
	}
	var in struct {
		Access string `json:"access"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "Data tidak valid")
		return
	}
	if id == c.UserID {
		writeErr(w, http.StatusBadRequest, "Anda tidak dapat mengubah akses akun sendiri")
		return
	}
	var name, role string
	if err := h.DB.QueryRow(ctx, `SELECT name, role FROM users WHERE id=$1`, id).Scan(&name, &role); err != nil {
		writeErr(w, http.StatusNotFound, "Pengguna tidak ditemukan")
		return
	}

	switch in.Access {
	case "inactive":
		_, _ = h.DB.Exec(ctx, `UPDATE users SET active=false WHERE id=$1`, id)
		h.audit(ctx, c.UserID, "menonaktifkan akun "+name)
	case "active":
		_, _ = h.DB.Exec(ctx, `UPDATE users SET active=true WHERE id=$1`, id)
		h.audit(ctx, c.UserID, "mengaktifkan kembali akun "+name)
	default:
		a, ok := accessMap[in.Access]
		if !ok || role == "mahasiswa" {
			writeErr(w, http.StatusBadRequest, "Peran tidak valid")
			return
		}
		_, _ = h.DB.Exec(ctx, `UPDATE users SET role=$1, position=$2, active=true WHERE id=$3`, a.Role, a.Position, id)
		h.audit(ctx, c.UserID, "mengubah hak akses "+name)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func parseDay(s string, plusDay bool) *time.Time {
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return nil
	}
	if plusDay {
		t = t.AddDate(0, 0, 1)
	}
	return &t
}

func (h *Handler) AuditLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rows, err := h.DB.Query(r.Context(), `
		SELECT id, actor_name, action, created_at FROM audit_logs
		WHERE ($1::timestamptz IS NULL OR created_at >= $1::timestamptz)
		  AND ($2::timestamptz IS NULL OR created_at < $2::timestamptz)
		  AND ($3::text = '' OR action ILIKE '%' || $3::text || '%' OR actor_name ILIKE '%' || $3::text || '%')
		ORDER BY created_at DESC, id DESC LIMIT 300`,
		parseDay(q.Get("from"), false), parseDay(q.Get("to"), true), strings.TrimSpace(q.Get("q")))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memuat audit log")
		return
	}
	defer rows.Close()
	type logRow struct {
		ID        int       `json:"id"`
		ActorName string    `json:"actor_name"`
		Action    string    `json:"action"`
		CreatedAt time.Time `json:"created_at"`
	}
	out := []logRow{}
	for rows.Next() {
		var l logRow
		if rows.Scan(&l.ID, &l.ActorName, &l.Action, &l.CreatedAt) == nil {
			out = append(out, l)
		}
	}
	writeJSON(w, http.StatusOK, out)
}
