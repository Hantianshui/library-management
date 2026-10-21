package library

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrNotFound        = errors.New("resource not found")
	ErrConflict        = errors.New("resource already exists")
	ErrInvalid         = errors.New("invalid input")
	ErrBookUnavailable = errors.New("book has no available copies")
	ErrMemberInactive  = errors.New("member is inactive")
	ErrLoanLimit       = errors.New("member active loan limit reached")
	ErrHasLoans        = errors.New("resource has loan history")
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(ctx context.Context, databaseURL string) (*Store, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}
	config.MaxConns = 20
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Migrate(ctx context.Context, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := s.pool.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("run migration %s: %w", name, err)
		}
	}
	return nil
}

func (s *Store) EnsureAdmin(ctx context.Context, username, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO users (username, password_hash) VALUES ($1, $2)
		ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash`, username, string(hash))
	return err
}

func (s *Store) Authenticate(ctx context.Context, username, password string) (User, error) {
	var user User
	var hash string
	err := s.pool.QueryRow(ctx,
		`SELECT id::text, username, password_hash FROM users WHERE username = $1`,
		username,
	).Scan(&user.ID, &user.Username, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return User{}, ErrNotFound
	}
	return user, nil
}

func (s *Store) UserByID(ctx context.Context, id string) (User, error) {
	var user User
	err := s.pool.QueryRow(ctx,
		`SELECT id::text, username FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Username)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return user, err
}

func (s *Store) Dashboard(ctx context.Context) (Dashboard, error) {
	var dashboard Dashboard
	err := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE active),
			COALESCE(SUM(total_copies) FILTER (WHERE active), 0),
			COALESCE(SUM(available_copies) FILTER (WHERE active), 0)
		FROM books`).Scan(&dashboard.BookTitles, &dashboard.TotalCopies, &dashboard.AvailableCopies)
	if err != nil {
		return dashboard, err
	}
	err = s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE returned_at IS NULL),
			COUNT(*) FILTER (WHERE returned_at IS NULL AND due_date < CURRENT_DATE)
		FROM loans`).Scan(&dashboard.ActiveLoans, &dashboard.OverdueLoans)
	if err != nil {
		return dashboard, err
	}
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM members WHERE active`).Scan(&dashboard.ActiveMembers); err != nil {
		return dashboard, err
	}
	loans, _, err := s.queryLoans(ctx, 1, 6, "", "")
	if err != nil {
		return dashboard, err
	}
	dashboard.RecentLoans = loans
	rows, err := s.pool.Query(ctx, `
		SELECT id, action, entity_type,
			COALESCE(detail->>'summary', action), created_at
		FROM audit_logs ORDER BY created_at DESC LIMIT 8`)
	if err != nil {
		return dashboard, err
	}
	defer rows.Close()
	dashboard.RecentActivity = []Activity{}
	for rows.Next() {
		var item Activity
		if err := rows.Scan(&item.ID, &item.Action, &item.EntityType, &item.Summary, &item.CreatedAt); err != nil {
			return dashboard, err
		}
		dashboard.RecentActivity = append(dashboard.RecentActivity, item)
	}
	return dashboard, rows.Err()
}

func (s *Store) ListBooks(ctx context.Context, page, pageSize int, search, status, category string) (Page[Book], error) {
	offset := (page - 1) * pageSize
	search = "%" + strings.TrimSpace(search) + "%"
	statusClause := ""
	switch status {
	case "available":
		statusClause = " AND b.active AND b.available_copies > 0"
	case "unavailable":
		statusClause = " AND b.active AND b.available_copies = 0"
	case "inactive":
		statusClause = " AND NOT b.active"
	}
	categoryClause := ""
	args := []any{search}
	if category != "" {
		args = append(args, category)
		categoryClause = fmt.Sprintf(" AND b.category = $%d", len(args))
	}
	var total int64
	countSQL := `SELECT COUNT(*) FROM books b
		WHERE (b.title ILIKE $1 OR b.author ILIKE $1 OR b.isbn ILIKE $1)` + statusClause + categoryClause
	if err := s.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return Page[Book]{}, err
	}
	args = append(args, pageSize, offset)
	query := `SELECT id::text, isbn, title, author, category, publisher, published_year,
			total_copies, available_copies, active, created_at, updated_at
		FROM books b
		WHERE (b.title ILIKE $1 OR b.author ILIKE $1 OR b.isbn ILIKE $1)` + statusClause + categoryClause +
		fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return Page[Book]{}, err
	}
	defer rows.Close()
	items := []Book{}
	for rows.Next() {
		book, err := scanBook(rows)
		if err != nil {
			return Page[Book]{}, err
		}
		items = append(items, book)
	}
	return Page[Book]{Items: items, Page: page, PageSize: pageSize, Total: total}, rows.Err()
}

func (s *Store) BookByID(ctx context.Context, id string) (Book, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id::text, isbn, title, author, category, publisher,
			published_year, total_copies, available_copies, active,
			created_at, updated_at
		FROM books
		WHERE id = $1`,
		id,
	)

	book, err := scanBook(row)
	return book, normalizeError(err)
}

func (s *Store) Categories(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT category FROM books WHERE category <> '' ORDER BY category`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		items = append(items, value)
	}
	return items, rows.Err()
}

func (s *Store) CreateBook(ctx context.Context, userID string, input BookInput) (Book, error) {
	var book Book
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			INSERT INTO books (isbn, title, author, category, publisher, published_year,
				total_copies, available_copies, active)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$7,$8)
			RETURNING id::text, isbn, title, author, category, publisher, published_year,
				total_copies, available_copies, active, created_at, updated_at`,
			input.ISBN, input.Title, input.Author, input.Category, input.Publisher,
			input.PublishedYear, input.TotalCopies, input.Active)
		var err error
		book, err = scanBook(row)
		if err != nil {
			return err
		}
		return audit(ctx, tx, userID, "book.created", "book", book.ID, "新增图书《"+book.Title+"》")
	})
	return book, normalizeError(err)
}

func (s *Store) UpdateBook(ctx context.Context, userID, id string, input BookInput) (Book, error) {
	var book Book
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		var borrowed int
		if err := tx.QueryRow(ctx,
			`SELECT total_copies - available_copies FROM books WHERE id = $1 FOR UPDATE`, id,
		).Scan(&borrowed); err != nil {
			return err
		}
		if input.TotalCopies < borrowed {
			return fmt.Errorf("%w: total copies cannot be less than %d borrowed copies", ErrInvalid, borrowed)
		}
		row := tx.QueryRow(ctx, `
			UPDATE books SET isbn=$2, title=$3, author=$4, category=$5, publisher=$6,
				published_year=$7, total_copies=$8, available_copies=$8-$9, active=$10, updated_at=now()
			WHERE id=$1
			RETURNING id::text, isbn, title, author, category, publisher, published_year,
				total_copies, available_copies, active, created_at, updated_at`,
			id, input.ISBN, input.Title, input.Author, input.Category, input.Publisher,
			input.PublishedYear, input.TotalCopies, borrowed, input.Active)
		var err error
		book, err = scanBook(row)
		if err != nil {
			return err
		}
		return audit(ctx, tx, userID, "book.updated", "book", book.ID, "更新图书《"+book.Title+"》")
	})
	return book, normalizeError(err)
}

func (s *Store) DeleteBook(ctx context.Context, userID, id string) error {
	return normalizeError(s.withTx(ctx, func(tx pgx.Tx) error {
		var title string
		if err := tx.QueryRow(ctx, `SELECT title FROM books WHERE id=$1`, id).Scan(&title); err != nil {
			return err
		}
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM loans WHERE book_id=$1)`, id).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return ErrHasLoans
		}
		if _, err := tx.Exec(ctx, `DELETE FROM books WHERE id=$1`, id); err != nil {
			return err
		}
		return audit(ctx, tx, userID, "book.deleted", "book", id, "删除图书《"+title+"》")
	}))
}

func (s *Store) ListMembers(ctx context.Context, page, pageSize int, search, status string) (Page[Member], error) {
	offset := (page - 1) * pageSize
	search = "%" + strings.TrimSpace(search) + "%"
	statusClause := ""
	if status == "active" {
		statusClause = " AND m.active"
	} else if status == "inactive" {
		statusClause = " AND NOT m.active"
	}
	var total int64
	countSQL := `SELECT COUNT(*) FROM members m
		WHERE (m.name ILIKE $1 OR m.member_no ILIKE $1 OR m.phone ILIKE $1 OR m.email ILIKE $1)` + statusClause
	if err := s.pool.QueryRow(ctx, countSQL, search).Scan(&total); err != nil {
		return Page[Member]{}, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT m.id::text, m.member_no, m.name, m.phone, m.email, m.active,
			COUNT(l.id) FILTER (WHERE l.returned_at IS NULL), m.created_at, m.updated_at
		FROM members m LEFT JOIN loans l ON l.member_id=m.id
		WHERE (m.name ILIKE $1 OR m.member_no ILIKE $1 OR m.phone ILIKE $1 OR m.email ILIKE $1)`+
		statusClause+`
		GROUP BY m.id ORDER BY m.created_at DESC LIMIT $2 OFFSET $3`, search, pageSize, offset)
	if err != nil {
		return Page[Member]{}, err
	}
	defer rows.Close()
	items := []Member{}
	for rows.Next() {
		var member Member
		if err := rows.Scan(&member.ID, &member.MemberNo, &member.Name, &member.Phone, &member.Email,
			&member.Active, &member.ActiveLoans, &member.CreatedAt, &member.UpdatedAt); err != nil {
			return Page[Member]{}, err
		}
		items = append(items, member)
	}
	return Page[Member]{Items: items, Page: page, PageSize: pageSize, Total: total}, rows.Err()
}

func (s *Store) CreateMember(ctx context.Context, userID string, input MemberInput) (Member, error) {
	var member Member
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO members (member_no, name, phone, email, active)
			VALUES ($1,$2,$3,$4,$5)
			RETURNING id::text, member_no, name, phone, email, active, 0, created_at, updated_at`,
			input.MemberNo, input.Name, input.Phone, input.Email, input.Active,
		).Scan(&member.ID, &member.MemberNo, &member.Name, &member.Phone, &member.Email,
			&member.Active, &member.ActiveLoans, &member.CreatedAt, &member.UpdatedAt)
		if err != nil {
			return err
		}
		return audit(ctx, tx, userID, "member.created", "member", member.ID, "新增读者 "+member.Name)
	})
	return member, normalizeError(err)
}

func (s *Store) UpdateMember(ctx context.Context, userID, id string, input MemberInput) (Member, error) {
	var member Member
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			UPDATE members SET member_no=$2, name=$3, phone=$4, email=$5, active=$6, updated_at=now()
			WHERE id=$1
			RETURNING id::text, member_no, name, phone, email, active,
				(SELECT COUNT(*) FROM loans WHERE member_id=$1 AND returned_at IS NULL),
				created_at, updated_at`,
			id, input.MemberNo, input.Name, input.Phone, input.Email, input.Active,
		).Scan(&member.ID, &member.MemberNo, &member.Name, &member.Phone, &member.Email,
			&member.Active, &member.ActiveLoans, &member.CreatedAt, &member.UpdatedAt)
		if err != nil {
			return err
		}
		return audit(ctx, tx, userID, "member.updated", "member", member.ID, "更新读者 "+member.Name)
	})
	return member, normalizeError(err)
}

func (s *Store) DeleteMember(ctx context.Context, userID, id string) error {
	return normalizeError(s.withTx(ctx, func(tx pgx.Tx) error {
		var name string
		if err := tx.QueryRow(ctx, `SELECT name FROM members WHERE id=$1`, id).Scan(&name); err != nil {
			return err
		}
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM loans WHERE member_id=$1)`, id).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return ErrHasLoans
		}
		if _, err := tx.Exec(ctx, `DELETE FROM members WHERE id=$1`, id); err != nil {
			return err
		}
		return audit(ctx, tx, userID, "member.deleted", "member", id, "删除读者 "+name)
	}))
}

func (s *Store) ListLoans(ctx context.Context, page, pageSize int, search, status string) (Page[Loan], error) {
	items, total, err := s.queryLoans(ctx, page, pageSize, search, status)
	return Page[Loan]{Items: items, Page: page, PageSize: pageSize, Total: total}, err
}

func (s *Store) queryLoans(ctx context.Context, page, pageSize int, search, status string) ([]Loan, int64, error) {
	offset := (page - 1) * pageSize
	search = "%" + strings.TrimSpace(search) + "%"
	statusClause := ""
	switch status {
	case "borrowed":
		statusClause = " AND l.returned_at IS NULL AND l.due_date >= CURRENT_DATE"
	case "overdue":
		statusClause = " AND l.returned_at IS NULL AND l.due_date < CURRENT_DATE"
	case "returned":
		statusClause = " AND l.returned_at IS NOT NULL"
	}
	where := ` WHERE (b.title ILIKE $1 OR b.isbn ILIKE $1 OR m.name ILIKE $1 OR m.member_no ILIKE $1)` + statusClause
	var total int64
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM loans l
		JOIN books b ON b.id=l.book_id JOIN members m ON m.id=l.member_id`+where, search).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT l.id::text, b.id::text, b.title, b.isbn, m.id::text, m.name, m.member_no,
			l.borrowed_at, l.due_date::text, l.returned_at,
			CASE WHEN l.returned_at IS NOT NULL THEN 'returned'
				WHEN l.due_date < CURRENT_DATE THEN 'overdue' ELSE 'borrowed' END
		FROM loans l JOIN books b ON b.id=l.book_id JOIN members m ON m.id=l.member_id`+
		where+` ORDER BY l.borrowed_at DESC LIMIT $2 OFFSET $3`, search, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []Loan{}
	for rows.Next() {
		var loan Loan
		if err := rows.Scan(&loan.ID, &loan.BookID, &loan.BookTitle, &loan.BookISBN,
			&loan.MemberID, &loan.MemberName, &loan.MemberNo, &loan.BorrowedAt,
			&loan.DueDate, &loan.ReturnedAt, &loan.Status); err != nil {
			return nil, 0, err
		}
		items = append(items, loan)
	}
	return items, total, rows.Err()
}

func (s *Store) CreateLoan(ctx context.Context, userID string, input LoanInput) (Loan, error) {
	var loan Loan
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		var bookTitle, bookISBN string
		var available int
		var bookActive bool
		if err := tx.QueryRow(ctx,
			`SELECT title, isbn, available_copies, active FROM books WHERE id=$1 FOR UPDATE`,
			input.BookID,
		).Scan(&bookTitle, &bookISBN, &available, &bookActive); err != nil {
			return err
		}
		if !bookActive || available < 1 {
			return ErrBookUnavailable
		}
		var memberName, memberNo string
		var memberActive bool
		if err := tx.QueryRow(ctx,
			`SELECT name, member_no, active FROM members WHERE id=$1 FOR UPDATE`,
			input.MemberID,
		).Scan(&memberName, &memberNo, &memberActive); err != nil {
			return err
		}
		if !memberActive {
			return ErrMemberInactive
		}
		var activeLoans int
		if err := tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM loans WHERE member_id=$1 AND returned_at IS NULL`,
			input.MemberID,
		).Scan(&activeLoans); err != nil {
			return err
		}
		if activeLoans >= 5 {
			return ErrLoanLimit
		}
		if _, err := tx.Exec(ctx,
			`UPDATE books SET available_copies=available_copies-1, updated_at=now() WHERE id=$1`,
			input.BookID,
		); err != nil {
			return err
		}
		err := tx.QueryRow(ctx, `
			INSERT INTO loans (book_id, member_id, due_date, created_by)
			VALUES ($1,$2,$3,$4)
			RETURNING id::text, borrowed_at`,
			input.BookID, input.MemberID, input.DueDate, userID,
		).Scan(&loan.ID, &loan.BorrowedAt)
		if err != nil {
			return err
		}
		loan.BookID, loan.BookTitle, loan.BookISBN = input.BookID, bookTitle, bookISBN
		loan.MemberID, loan.MemberName, loan.MemberNo = input.MemberID, memberName, memberNo
		loan.DueDate, loan.Status = input.DueDate, "borrowed"
		return audit(ctx, tx, userID, "loan.created", "loan", loan.ID,
			"借出《"+bookTitle+"》给 "+memberName)
	})
	return loan, normalizeError(err)
}

func (s *Store) ReturnLoan(ctx context.Context, userID, id string) (Loan, error) {
	var loan Loan
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			SELECT l.id::text, b.id::text, b.title, b.isbn, m.id::text, m.name, m.member_no,
				l.borrowed_at, l.due_date::text, l.returned_at,
				CASE WHEN l.returned_at IS NOT NULL THEN 'returned'
					WHEN l.due_date < CURRENT_DATE THEN 'overdue' ELSE 'borrowed' END
			FROM loans l JOIN books b ON b.id=l.book_id JOIN members m ON m.id=l.member_id
			WHERE l.id=$1 FOR UPDATE OF l`, id,
		).Scan(&loan.ID, &loan.BookID, &loan.BookTitle, &loan.BookISBN, &loan.MemberID,
			&loan.MemberName, &loan.MemberNo, &loan.BorrowedAt, &loan.DueDate,
			&loan.ReturnedAt, &loan.Status)
		if err != nil {
			return err
		}
		if loan.ReturnedAt != nil {
			return nil
		}
		now := time.Now()
		if _, err := tx.Exec(ctx, `UPDATE loans SET returned_at=$2 WHERE id=$1`, id, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE books SET available_copies=available_copies+1, updated_at=now() WHERE id=$1`,
			loan.BookID,
		); err != nil {
			return err
		}
		loan.ReturnedAt, loan.Status = &now, "returned"
		return audit(ctx, tx, userID, "loan.returned", "loan", loan.ID,
			"归还《"+loan.BookTitle+"》")
	})
	return loan, normalizeError(err)
}

func (s *Store) withTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func audit(ctx context.Context, tx pgx.Tx, userID, action, entityType, entityID, summary string) error {
	detail, _ := json.Marshal(map[string]string{"summary": summary})
	_, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (user_id, action, entity_type, entity_id, detail)
		VALUES ($1,$2,$3,$4,$5)`, userID, action, entityType, entityID, detail)
	return err
}

type bookScanner interface {
	Scan(dest ...any) error
}

func scanBook(row bookScanner) (Book, error) {
	var book Book
	err := row.Scan(&book.ID, &book.ISBN, &book.Title, &book.Author, &book.Category,
		&book.Publisher, &book.PublishedYear, &book.TotalCopies, &book.AvailableCopies,
		&book.Active, &book.CreatedAt, &book.UpdatedAt)
	return book, err
}

func normalizeError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrConflict
		case "23503":
			return ErrHasLoans
		case "23514", "22P02":
			return ErrInvalid
		}
	}
	return err
}
