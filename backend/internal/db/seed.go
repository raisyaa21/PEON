package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const tplAktif = `Yang bertanda tangan di bawah ini, {{jabatan}} Fakultas Ilmu Komputer Universitas Brawijaya, menerangkan bahwa:

Nama : {{nama}}
NIM : {{nim}}
Program Studi : {{prodi}}

adalah benar mahasiswa aktif pada semester {{semester}}.

Surat keterangan ini dibuat untuk keperluan: {{keperluan}}.

Demikian surat keterangan ini dibuat untuk dipergunakan sebagaimana mestinya.`

const tplPengantar = `Dengan hormat,

Sehubungan dengan kegiatan akademik mahasiswa kami berikut ini:

Nama : {{nama}}
NIM : {{nim}}
Program Studi : {{prodi}}

Kami mohon bantuan Bapak/Ibu agar mahasiswa tersebut dapat diterima untuk melaksanakan kegiatan: {{keperluan}}.

Demikian surat ini kami sampaikan. Atas perhatian dan kerja sama Bapak/Ibu, kami ucapkan terima kasih.`

const tplIzin = `Yang bertanda tangan di bawah ini, {{jabatan}} Fakultas Ilmu Komputer Universitas Brawijaya, menerangkan bahwa mahasiswa berikut:

Nama : {{nama}}
NIM : {{nim}}
Program Studi : {{prodi}}

mengajukan {{jenis}} dengan keterangan: {{keperluan}}.

Permohonan ini telah melalui pertimbangan Dosen Pembimbing Akademik dan dinyatakan memenuhi ketentuan yang berlaku.

Demikian surat ini dibuat untuk dipergunakan sebagaimana mestinya.`

func Seed(ctx context.Context, pool *pgxpool.Pool) error {
	var n int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	pw := string(hash)

	var dosenID int
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (identifier, name, password_hash, role, position)
		VALUES ('197803152005','Dr. Ratna Sari, M.Kom.',$1,'dosen','Dosen Pembimbing Akademik (Dosen PA)') RETURNING id`, pw).Scan(&dosenID); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO users (identifier, name, password_hash, role, position)
		VALUES ('198501012010','Rina Wulandari',$1,'staf','Staf Administrasi & Operasional')`, pw); err != nil {
		return err
	}

	type stu struct {
		NIM, Name string
		Sem       int
		IPS       []float64
	}
	students := []stu{
		{"255150400111001", "Budi Santoso", 5, []float64{3.45, 3.52, 3.60, 3.58}},
		{"255150400111002", "Siti Aisyah", 5, []float64{3.70, 3.65, 3.72, 3.80}},
		{"255150400111003", "Fajar Pratama", 5, []float64{3.10, 3.25, 3.30, 3.42}},
	}
	for _, s := range students {
		var sid int
		if err := pool.QueryRow(ctx, `
			INSERT INTO users (identifier, name, password_hash, role, advisor_id, study_program, semester)
			VALUES ($1,$2,$3,'mahasiswa',$4,'Sistem Informasi',$5) RETURNING id`,
			s.NIM, s.Name, pw, dosenID, s.Sem).Scan(&sid); err != nil {
			return err
		}
		for i, v := range s.IPS {
			if _, err := pool.Exec(ctx,
				`INSERT INTO academic_records (student_id, semester, ips, sks) VALUES ($1,$2,$3,$4)`,
				sid, i+1, v, 20+i%3); err != nil {
				return err
			}
		}
	}

	type lt struct {
		Code, Name, Desc string
		Rec              bool
		Tpl              string
		Reqs             []string
	}
	letters := []lt{
		{"SKAK", "Surat Keterangan Aktif Kuliah", "Bukti status aktif untuk beasiswa, tunjangan, atau instansi lain.", false, tplAktif,
			[]string{"Scan KTM", "Bukti Pembayaran UKT"}},
		{"SPMG", "Surat Pengantar Magang/PKL", "Pengantar resmi fakultas untuk magang atau praktik kerja lapangan.", false, tplPengantar,
			[]string{"Scan KTM", "KHS Semester Terakhir", "Lembar Persetujuan Instansi"}},
		{"SIRS", "Surat Izin Riset/Penelitian Skripsi", "Izin pengambilan data penelitian skripsi pada instansi luar.", true, tplPengantar,
			[]string{"Scan KTM", "KHS Semester Terakhir"}},
		{"SCUT", "Surat Permohonan Cuti Akademik", "Permohonan cuti studi untuk satu semester.", true, tplIzin,
			[]string{"Scan KTM", "Bukti Pembayaran UKT", "Rekomendasi Dosen PA"}},
		{"SDIS", "Surat Dispensasi Kuliah", "Izin tidak mengikuti perkuliahan karena kegiatan resmi.", true, tplIzin,
			[]string{"Scan KTM", "Surat Undangan/Bukti Kegiatan"}},
		{"SPDR", "Surat Pengunduran Diri", "Permohonan pengunduran diri sebagai mahasiswa.", false, tplIzin,
			[]string{"Scan KTM", "KHS Semester Terakhir"}},
	}
	for _, l := range letters {
		var id int
		if err := pool.QueryRow(ctx, `
			INSERT INTO letter_types (code, name, description, requires_recommendation, template_body)
			VALUES ($1,$2,$3,$4,$5) RETURNING id`,
			l.Code, l.Name, l.Desc, l.Rec, l.Tpl).Scan(&id); err != nil {
			return err
		}
		for _, r := range l.Reqs {
			if _, err := pool.Exec(ctx,
				`INSERT INTO letter_requirements (letter_type_id, name) VALUES ($1,$2)`, id, r); err != nil {
				return err
			}
		}
	}

	settings := [][2]string{
		{"kop_1", "KEMENTERIAN PENDIDIKAN TINGGI, SAINS, DAN TEKNOLOGI"},
		{"kop_2", "UNIVERSITAS BRAWIJAYA"},
		{"kop_3", "FAKULTAS ILMU KOMPUTER"},
		{"address", "Jl. Veteran No. 8, Malang, Jawa Timur 65145"},
		{"city", "Malang"},
		{"signer_name", "Dr. Eng. Irwan Santoso, S.Kom., M.T."},
		{"signer_title", "Ketua Program Studi"},
	}
	for _, s := range settings {
		if _, err := pool.Exec(ctx, `INSERT INTO settings (key, value) VALUES ($1,$2)`, s[0], s[1]); err != nil {
			return err
		}
	}
	return nil
}
