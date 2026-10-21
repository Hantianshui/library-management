package library

import "time"

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type Book struct {
	ID              string    `json:"id"`
	ISBN            string    `json:"isbn"`
	Title           string    `json:"title"`
	Author          string    `json:"author"`
	Category        string    `json:"category"`
	Publisher       string    `json:"publisher"`
	PublishedYear   *int      `json:"publishedYear"`
	TotalCopies     int       `json:"totalCopies"`
	AvailableCopies int       `json:"availableCopies"`
	Active          bool      `json:"active"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type Member struct {
	ID          string    `json:"id"`
	MemberNo    string    `json:"memberNo"`
	Name        string    `json:"name"`
	Phone       string    `json:"phone"`
	Email       string    `json:"email"`
	Active      bool      `json:"active"`
	ActiveLoans int       `json:"activeLoans"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Loan struct {
	ID         string     `json:"id"`
	BookID     string     `json:"bookId"`
	BookTitle  string     `json:"bookTitle"`
	BookISBN   string     `json:"bookIsbn"`
	MemberID   string     `json:"memberId"`
	MemberName string     `json:"memberName"`
	MemberNo   string     `json:"memberNo"`
	BorrowedAt time.Time  `json:"borrowedAt"`
	DueDate    string     `json:"dueDate"`
	ReturnedAt *time.Time `json:"returnedAt"`
	Status     string     `json:"status"`
}

type Activity struct {
	ID         int64     `json:"id"`
	Action     string    `json:"action"`
	EntityType string    `json:"entityType"`
	Summary    string    `json:"summary"`
	CreatedAt  time.Time `json:"createdAt"`
}

type Dashboard struct {
	BookTitles      int        `json:"bookTitles"`
	TotalCopies     int        `json:"totalCopies"`
	AvailableCopies int        `json:"availableCopies"`
	ActiveLoans     int        `json:"activeLoans"`
	OverdueLoans    int        `json:"overdueLoans"`
	ActiveMembers   int        `json:"activeMembers"`
	RecentLoans     []Loan     `json:"recentLoans"`
	RecentActivity  []Activity `json:"recentActivity"`
}

type BookInput struct {
	ISBN          string `json:"isbn"`
	Title         string `json:"title"`
	Author        string `json:"author"`
	Category      string `json:"category"`
	Publisher     string `json:"publisher"`
	PublishedYear *int   `json:"publishedYear"`
	TotalCopies   int    `json:"totalCopies"`
	Active        bool   `json:"active"`
}

type MemberInput struct {
	MemberNo string `json:"memberNo"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Active   bool   `json:"active"`
}

type LoanInput struct {
	BookID   string `json:"bookId"`
	MemberID string `json:"memberId"`
	DueDate  string `json:"dueDate"`
}

type Page[T any] struct {
	Items    []T   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
	Total    int64 `json:"total"`
}
