package handlers

import (
	"bytes"
	"strconv"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

type letterData struct {
	Kop         []string
	Address     string
	Title       string
	LetterNo    string
	Body        string
	City        string
	Date        time.Time
	SignerName  string
	SignerTitle string
	Final       bool
	QRPNG       []byte
	VerifyURL   string
	DocHash     string
}

func renderLetterPDF(d letterData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(25, 18, 25)
	pdf.SetAutoPageBreak(true, 18)
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.AddPage()

	if !d.Final {
		pdf.TransformBegin()
		pdf.TransformRotate(45, 105, 150)
		pdf.SetFont("Helvetica", "B", 100)
		pdf.SetTextColor(236, 238, 244)
		pdf.Text(45, 165, "DRAF")
		pdf.TransformEnd()
		pdf.SetTextColor(0, 0, 0)
	}

	// kop surat
	sizes := []float64{10, 13, 12}
	styles := []string{"", "B", "B"}
	for i, l := range d.Kop {
		if i >= 3 {
			break
		}
		pdf.SetFont("Helvetica", styles[i], sizes[i])
		pdf.CellFormat(0, 6, tr(l), "", 1, "C", false, 0, "")
	}
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(0, 5, tr(d.Address), "", 1, "C", false, 0, "")
	y := pdf.GetY() + 2
	pdf.SetLineWidth(0.8)
	pdf.Line(25, y, 185, y)
	pdf.SetLineWidth(0.2)
	pdf.Line(25, y+1, 185, y+1)
	pdf.SetY(y + 8)

	// judul + nomor
	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(0, 7, tr(strings.ToUpper(d.Title)), "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 11)
	pdf.CellFormat(0, 6, tr("Nomor: "+d.LetterNo), "", 1, "C", false, 0, "")
	pdf.Ln(8)

	// isi
	pdf.SetFont("Helvetica", "", 11)
	pdf.MultiCell(0, 6.5, tr(d.Body), "", "L", false)
	pdf.Ln(10)

	// blok tanda tangan
	if pdf.GetY() > 205 {
		pdf.AddPage()
	}
	x := 115.0
	pdf.SetX(x)
	pdf.CellFormat(70, 6, tr(d.City+", "+dateID(d.Date)), "", 1, "L", false, 0, "")
	pdf.SetX(x)
	pdf.MultiCell(70, 6, tr(d.SignerTitle+","), "", "L", false)
	sigY := pdf.GetY() + 2

	if d.Final && len(d.QRPNG) > 0 {
		opt := fpdf.ImageOptions{ImageType: "PNG"}
		pdf.RegisterImageOptionsReader("qr", opt, bytes.NewReader(d.QRPNG))
		pdf.ImageOptions("qr", x, sigY, 28, 28, false, opt, 0, "")
		pdf.SetXY(x, sigY+29)
		pdf.SetFont("Helvetica", "I", 7.5)
		pdf.CellFormat(70, 4, tr("Ditandatangani secara elektronik"), "", 1, "L", false, 0, "")
	} else {
		pdf.SetXY(x, sigY+10)
		pdf.SetFont("Helvetica", "I", 9)
		pdf.SetTextColor(130, 130, 130)
		pdf.CellFormat(70, 5, tr("(menunggu tanda tangan elektronik)"), "", 1, "L", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.SetXY(x, sigY+29)
	}
	pdf.SetX(x)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(70, 6, tr(d.SignerName), "", 1, "L", false, 0, "")

	if d.Final {
		pdf.Ln(6)
		hash := d.DocHash
		if len(hash) > 16 {
			hash = hash[:16]
		}
		pdf.SetX(25)
		pdf.SetFont("Helvetica", "", 7.5)
		pdf.SetTextColor(100, 100, 100)
		pdf.MultiCell(0, 4, tr("Dokumen ini ditandatangani secara elektronik. Periksa keaslian dokumen melalui kode QR atau tautan: "+
			d.VerifyURL+" (hash: "+hash+")"), "", "L", false)
		pdf.SetTextColor(0, 0, 0)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func renderReportPDF(r ReportData, period string) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 18, 20)
	pdf.SetAutoPageBreak(true, 18)
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 15)
	pdf.CellFormat(0, 8, tr("Laporan Rekapitulasi Persuratan Akademik"), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, tr("Periode: "+period), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, tr("Dicetak: "+dateID(time.Now())), "", 1, "L", false, 0, "")
	pdf.Ln(4)

	section := func(t string) {
		pdf.Ln(2)
		pdf.SetFont("Helvetica", "B", 11)
		pdf.CellFormat(0, 8, tr(t), "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
	}
	row := func(a, b string, bold bool) {
		if bold {
			pdf.SetFont("Helvetica", "B", 10)
		} else {
			pdf.SetFont("Helvetica", "", 10)
		}
		pdf.CellFormat(120, 7, tr(a), "1", 0, "L", false, 0, "")
		pdf.CellFormat(50, 7, tr(b), "1", 1, "R", false, 0, "")
	}
	pct := func(n int) string {
		if r.Total == 0 {
			return "0 (0%)"
		}
		return strconv.Itoa(n) + " (" + strconv.Itoa(n*100/r.Total) + "%)"
	}

	section("Ringkasan")
	row("Total pengajuan", strconv.Itoa(r.Total), true)

	section("Pengajuan per jenis surat")
	if len(r.ByType) == 0 {
		row("(tidak ada data)", "-", false)
	}
	for _, t := range r.ByType {
		row(t.Name, strconv.Itoa(t.Count), false)
	}

	section("Status penyelesaian")
	row("Selesai diterbitkan", pct(r.Completed), false)
	row("Sedang diproses", pct(r.Processing), false)
	row("Ditolak", pct(r.Rejected), false)

	section("Volume pengajuan per bulan (" + strconv.Itoa(r.Year) + ")")
	for i, n := range r.Monthly {
		row(monthID[i+1], strconv.Itoa(n), false)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
