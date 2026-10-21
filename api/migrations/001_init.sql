CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    username varchar(64) NOT NULL UNIQUE,
    password_hash varchar(255) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS books (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    isbn varchar(32) NOT NULL UNIQUE,
    title varchar(255) NOT NULL,
    author varchar(255) NOT NULL,
    category varchar(100) NOT NULL DEFAULT '',
    publisher varchar(255) NOT NULL DEFAULT '',
    published_year int CHECK (published_year IS NULL OR published_year BETWEEN 1000 AND 9999),
    total_copies int NOT NULL CHECK (total_copies >= 0),
    available_copies int NOT NULL CHECK (available_copies >= 0 AND available_copies <= total_copies),
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS members (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    member_no varchar(32) NOT NULL UNIQUE,
    name varchar(100) NOT NULL,
    phone varchar(32) NOT NULL DEFAULT '',
    email varchar(255) NOT NULL DEFAULT '',
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS loans (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    book_id uuid NOT NULL REFERENCES books(id),
    member_id uuid NOT NULL REFERENCES members(id),
    borrowed_at timestamptz NOT NULL DEFAULT now(),
    due_date date NOT NULL,
    returned_at timestamptz,
    created_by uuid NOT NULL REFERENCES users(id),
    CHECK (returned_at IS NULL OR returned_at >= borrowed_at)
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id bigserial PRIMARY KEY,
    user_id uuid REFERENCES users(id),
    action varchar(64) NOT NULL,
    entity_type varchar(64) NOT NULL,
    entity_id uuid,
    detail jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_books_title ON books (title);
CREATE INDEX IF NOT EXISTS idx_members_name ON members (name);
CREATE INDEX IF NOT EXISTS idx_loans_book_id ON loans (book_id);
CREATE INDEX IF NOT EXISTS idx_loans_member_id ON loans (member_id);
CREATE INDEX IF NOT EXISTS idx_loans_active_due_date ON loans (due_date) WHERE returned_at IS NULL;

INSERT INTO books (isbn, title, author, category, publisher, published_year, total_copies, available_copies)
VALUES
    ('9787115428028', 'Go语言实战', 'William Kennedy', '计算机', '人民邮电出版社', 2017, 4, 4),
    ('9787115546081', '深入理解计算机系统', 'Randal E. Bryant', '计算机', '机械工业出版社', 2021, 3, 3),
    ('9787544253994', '百年孤独', '加西亚·马尔克斯', '文学', '南海出版公司', 2011, 5, 5)
ON CONFLICT (isbn) DO NOTHING;

INSERT INTO members (member_no, name, phone, email)
VALUES
    ('R2026001', '陈晓宁', '13800000001', 'xiaoning@example.com'),
    ('R2026002', '周明远', '13800000002', 'mingyuan@example.com')
ON CONFLICT (member_no) DO NOTHING;
