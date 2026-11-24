package handlers

import (
	"net/http"
	"strconv"
	"strings"
)

type LetterType struct {
	ID                     int           `json:"id"`
	Code                   string        `json:"code"`
	Name                   string        `json:"name"`
	Description            string        `json:"description"`
	RequiresRecommendation bool          `json:"requires_recommendation"`
	Requirements           []Requirement `json:"requirements"`
}

const letterTypeSQL = `
SELECT lt.id, lt.code, lt.name, lt.description, lt.requires_recommendation,
       COALESCE(json_agg(json_build_object('id', lr.id, 'name', lr.name) ORDER BY lr.id)
                FILTER (WHERE lr.id IS NOT NULL), '[]'::json)
FROM letter_types lt
LEFT JOIN letter_requirements lr ON lr.letter_type_id = lt.id
WHERE lt.active
  AND ($2::int = 0 OR lt.id = $2::int)
  AND ($1::text = '' OR lt.name ILIKE '%' || $1::text || '%' OR lt.description ILIKE '%' || $1::text || '%')
GROUP BY lt.id
ORDER BY lt.id`

func (h *Handler) queryLetterTypes(r *http.Request, q string, id int) ([]LetterType, error) {
	rows, err := h.DB.Query(r.Context(), letterTypeSQL, strings.TrimSpace(q), id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LetterType{}
	for rows.Next() {
		var l LetterType
		if err := rows.Scan(&l.ID, &l.Code, &l.Name, &l.Description, &l.RequiresRecommendation, &l.Requirements); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// US-02
func (h *Handler) ListLetterTypes(w http.ResponseWriter, r *http.Request) {
	items, err := h.queryLetterTypes(r, r.URL.Query().Get("q"), 0)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memuat jenis surat")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) GetLetterType(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeErr(w, http.StatusNotFound, "Jenis surat tidak ditemukan")
		return
	}
	items, err := h.queryLetterTypes(r, "", id)
	if err != nil || len(items) == 0 {
		writeErr(w, http.StatusNotFound, "Jenis surat tidak ditemukan")
		return
	}
	writeJSON(w, http.StatusOK, items[0])
}
