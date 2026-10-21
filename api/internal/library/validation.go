package library

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

func ValidateBook(input *BookInput) error {
	input.ISBN = strings.TrimSpace(input.ISBN)
	input.Title = strings.TrimSpace(input.Title)
	input.Author = strings.TrimSpace(input.Author)
	input.Category = strings.TrimSpace(input.Category)
	input.Publisher = strings.TrimSpace(input.Publisher)
	if input.ISBN == "" || input.Title == "" || input.Author == "" {
		return fmt.Errorf("%w: ISBN、书名和作者不能为空", ErrInvalid)
	}
	if input.TotalCopies < 0 {
		return fmt.Errorf("%w: 馆藏数量不能小于 0", ErrInvalid)
	}
	if input.PublishedYear != nil && (*input.PublishedYear < 1000 || *input.PublishedYear > 9999) {
		return fmt.Errorf("%w: 出版年份格式不正确", ErrInvalid)
	}
	return nil
}

func ValidateMember(input *MemberInput) error {
	input.MemberNo = strings.TrimSpace(input.MemberNo)
	input.Name = strings.TrimSpace(input.Name)
	input.Phone = strings.TrimSpace(input.Phone)
	input.Email = strings.TrimSpace(input.Email)
	if input.MemberNo == "" || input.Name == "" {
		return fmt.Errorf("%w: 读者编号和姓名不能为空", ErrInvalid)
	}
	if input.Email != "" && !emailPattern.MatchString(input.Email) {
		return fmt.Errorf("%w: 邮箱格式不正确", ErrInvalid)
	}
	return nil
}

func ValidateLoan(input *LoanInput) error {
	input.BookID = strings.TrimSpace(input.BookID)
	input.MemberID = strings.TrimSpace(input.MemberID)
	input.DueDate = strings.TrimSpace(input.DueDate)
	if input.BookID == "" || input.MemberID == "" || input.DueDate == "" {
		return fmt.Errorf("%w: 图书、读者和应还日期不能为空", ErrInvalid)
	}
	due, err := time.Parse(time.DateOnly, input.DueDate)
	if err != nil {
		return fmt.Errorf("%w: 应还日期格式不正确", ErrInvalid)
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if due.Before(today) {
		return fmt.Errorf("%w: 应还日期不能早于今天", ErrInvalid)
	}
	return nil
}
