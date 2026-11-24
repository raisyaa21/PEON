CREATE TABLE IF NOT EXISTS users (
  id            SERIAL PRIMARY KEY,
  identifier    VARCHAR(30) UNIQUE NOT NULL,
  name          VARCHAR(120) NOT NULL,
  password_hash TEXT NOT NULL,
  role          VARCHAR(10) NOT NULL CHECK (role IN ('mahasiswa','dosen','staf')),
  position      VARCHAR(100) NOT NULL DEFAULT '',
  active        BOOLEAN NOT NULL DEFAULT true,
  advisor_id    INT REFERENCES users(id),
  study_program VARCHAR(80) NOT NULL DEFAULT '',
  semester      INT NOT NULL DEFAULT 0,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS academic_records (
  student_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  semester   INT NOT NULL,
  ips        DOUBLE PRECISION NOT NULL,
  sks        INT NOT NULL,
  PRIMARY KEY (student_id, semester)
);

CREATE TABLE IF NOT EXISTS letter_types (
  id                      SERIAL PRIMARY KEY,
  code                    VARCHAR(20) UNIQUE NOT NULL,
  name                    VARCHAR(120) NOT NULL,
  description             TEXT NOT NULL DEFAULT '',
  active                  BOOLEAN NOT NULL DEFAULT true,
  requires_recommendation BOOLEAN NOT NULL DEFAULT false,
  number_format           VARCHAR(120) NOT NULL DEFAULT '{NO}/{KODE}/{JUR}/{BLN}/{THN}',
  template_body           TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS letter_requirements (
  id             SERIAL PRIMARY KEY,
  letter_type_id INT NOT NULL REFERENCES letter_types(id) ON DELETE CASCADE,
  name           VARCHAR(120) NOT NULL
);

CREATE TABLE IF NOT EXISTS letter_counters (
  letter_type_id INT NOT NULL REFERENCES letter_types(id) ON DELETE CASCADE,
  year           INT NOT NULL,
  last_no        INT NOT NULL DEFAULT 0,
  PRIMARY KEY (letter_type_id, year)
);

CREATE SEQUENCE IF NOT EXISTS ticket_seq;

CREATE TABLE IF NOT EXISTS tickets (
  id             SERIAL PRIMARY KEY,
  ticket_no      VARCHAR(30) UNIQUE NOT NULL,
  student_id     INT NOT NULL REFERENCES users(id),
  letter_type_id INT NOT NULL REFERENCES letter_types(id),
  purpose        TEXT NOT NULL,
  status         VARCHAR(30) NOT NULL DEFAULT 'submitted',
  note           TEXT NOT NULL DEFAULT '',
  verifier_unit  VARCHAR(80) NOT NULL DEFAULT '',
  advisor_status VARCHAR(12) NOT NULL DEFAULT '',
  advisor_note   TEXT NOT NULL DEFAULT '',
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_tickets_student ON tickets(student_id);
CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status);

CREATE TABLE IF NOT EXISTS ticket_documents (
  id             SERIAL PRIMARY KEY,
  ticket_id      INT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
  requirement_id INT NOT NULL REFERENCES letter_requirements(id),
  file_name      VARCHAR(255) NOT NULL,
  file_path      TEXT NOT NULL,
  mime           VARCHAR(50) NOT NULL,
  size_bytes     BIGINT NOT NULL,
  version        INT NOT NULL DEFAULT 1,
  is_current     BOOLEAN NOT NULL DEFAULT true,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ticket_history (
  id         SERIAL PRIMARY KEY,
  ticket_id  INT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
  status     VARCHAR(30) NOT NULL,
  unit       VARCHAR(80) NOT NULL DEFAULT '',
  note       TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS letters (
  id          SERIAL PRIMARY KEY,
  ticket_id   INT UNIQUE NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
  letter_no   VARCHAR(100) NOT NULL,
  verify_code VARCHAR(40) UNIQUE NOT NULL,
  draft_path  TEXT NOT NULL DEFAULT '',
  final_path  TEXT NOT NULL DEFAULT '',
  signed_by   VARCHAR(120) NOT NULL DEFAULT '',
  doc_hash    VARCHAR(64) NOT NULL DEFAULT '',
  issued_at   TIMESTAMPTZ,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS messages (
  id          SERIAL PRIMARY KEY,
  student_id  INT NOT NULL REFERENCES users(id),
  sender_id   INT NOT NULL REFERENCES users(id),
  sender_role VARCHAR(10) NOT NULL,
  body        TEXT NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_messages_student ON messages(student_id, created_at);

CREATE TABLE IF NOT EXISTS audit_logs (
  id         SERIAL PRIMARY KEY,
  actor_id   INT,
  actor_name VARCHAR(120) NOT NULL DEFAULT '',
  action     TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_logs(created_at);

CREATE TABLE IF NOT EXISTS settings (
  key   VARCHAR(40) PRIMARY KEY,
  value TEXT NOT NULL DEFAULT ''
);