package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"rsi-surat/internal/middleware"
)

type NameCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type ReportData struct {
	Month      int         `json:"month"`
	Year       int         `json:"year"`
	Total      int         `json:"total"`
	ByType     []NameCount `json:"by_type"`
	Completed  int         `json:"completed"`
	Processing int         `json:"processing"`
	Rejected   int         `json:"rejected"`
	Monthly    []int       `json:"monthly"`
}

func reportParams(r *http.Request) (int, int) {
	now := time.Now()
	month, year := int(now.Month()), now.Year()
	q := r.URL.Query()
	if v := q.Get("month"); v != "" {
		if m, err := strconv.Atoi(v); err == nil && m >= 0 && m <= 12 {
			month = m
		}
	}
	if v := q.Get("year"); v != "" {
		if y, err := strconv.Atoi(v); err == nil && y > 2000 && y < 2100 {
			year = y
		}
	}
	return month, year
}

func (h *Handler) buildReport(ctx context.Context, month, year int) (ReportData, error) {
	rep := ReportData{Month: month, Year: year, ByType: []NameCount{}, Monthly: make([]int, 12)}

	from := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	to := from.AddDate(1, 0, 0)
	if month >= 1 {
		from = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
		to = from.AddDate(0, 1, 0)
	}

	if err := h.DB.QueryRow(ctx, `
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE status='completed'),
		       COUNT(*) FILTER (WHERE status='rejected'),
		       COUNT(*) FILTER (WHERE status NOT IN ('completed','rejected'))
		FROM tickets WHERE created_at >= $1 AND created_at < $2`, from, to).
		Scan(&rep.Total, &rep.Completed, &rep.Rejected, &rep.Processing); err != nil {
		return rep, err
	}

	rows, err := h.DB.Query(ctx, `
		SELECT lt.name, COUNT(*) FROM tickets t JOIN letter_types lt ON lt.id = t.letter_type_id
		WHERE t.created_at >= $1 AND t.created_at < $2
		GROUP BY lt.name ORDER BY COUNT(*) DESC, lt.name`, from, to)
	if err != nil {
		return rep, err
	}
	for rows.Next() {
		var n NameCount
		if rows.Scan(&n.Name, &n.Count) == nil {
			rep.ByType = append(rep.ByType, n)
		}
	}
	rows.Close()

	yFrom := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	mrows, err := h.DB.Query(ctx, `
		SELECT EXTRACT(MONTH FROM created_at)::int, COUNT(*) FROM tickets
		WHERE created_at >= $1 AND created_at < $2 GROUP BY 1`, yFrom, yFrom.AddDate(1, 0, 0))
	if err != nil {
		return rep, err
	}
	defer mrows.Close()
	for mrows.Next() {
		var m, n int
		if mrows.Scan(&m, &n) == nil && m >= 1 && m <= 12 {
			rep.Monthly[m-1] = n
		}
	}
	return rep, nil
}

// US-15
func (h *Handler) Report(w http.ResponseWriter, r *http.Request) {
	month, year := reportParams(r)
	rep, err := h.buildReport(r.Context(), month, year)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memuat laporan")
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func (h *Handler) ReportExport(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	month, year := reportParams(r)
	rep, err := h.buildReport(r.Context(), month, year)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal memuat laporan")
		return
	}
	period := "Tahun " + strconv.Itoa(year)
	if month >= 1 {
		period = fmt.Sprintf("%s %d", monthID[month], year)
	}
	pdfBytes, err := renderReportPDF(rep, period)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Gagal membuat PDF laporan")
		return
	}
	h.audit(r.Context(), c.UserID, "mengekspor laporan rekapitulasi "+period)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="laporan-persuratan.pdf"`)
	_, _ = w.Write(pdfBytes)
}
