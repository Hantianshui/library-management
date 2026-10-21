package library

import (
	"errors"
	"testing"
	"time"
)

func TestValidateBook(t *testing.T) {
	t.Parallel()
	valid := BookInput{ISBN: " 978-7 ", Title: " Go ", Author: " Author ", TotalCopies: 2, Active: true}
	if err := ValidateBook(&valid); err != nil {
		t.Fatalf("ValidateBook() error = %v", err)
	}
	if valid.ISBN != "978-7" || valid.Title != "Go" {
		t.Fatalf("ValidateBook() did not trim input: %#v", valid)
	}

	invalid := BookInput{ISBN: "1", Title: "book", Author: "author", TotalCopies: -1}
	if err := ValidateBook(&invalid); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ValidateBook() error = %v, want ErrInvalid", err)
	}
}

func TestValidateMember(t *testing.T) {
	t.Parallel()
	input := MemberInput{MemberNo: "R1", Name: "Reader", Email: "bad-email"}
	if err := ValidateMember(&input); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ValidateMember() error = %v, want ErrInvalid", err)
	}
}

func TestValidateLoan(t *testing.T) {
	t.Parallel()
	input := LoanInput{
		BookID:   "book",
		MemberID: "member",
		DueDate:  time.Now().AddDate(0, 0, 7).Format(time.DateOnly),
	}
	if err := ValidateLoan(&input); err != nil {
		t.Fatalf("ValidateLoan() error = %v", err)
	}

	past := input
	past.DueDate = time.Now().AddDate(0, 0, -1).Format(time.DateOnly)
	if err := ValidateLoan(&past); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ValidateLoan() error = %v, want ErrInvalid", err)
	}
}
