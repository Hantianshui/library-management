package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"library-management/api/internal/library"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
)

const sessionCookie = "library_session"

type userKey struct{}

type Server struct {
	store        *library.Store
	jwtSecret    []byte
	cookieSecure bool
	logger       *slog.Logger
}

func New(store *library.Store, jwtSecret string, cookieSecure bool, logger *slog.Logger) *Server {
	return &Server{
		store:        store,
		jwtSecret:    []byte(jwtSecret),
		cookieSecure: cookieSecure,
		logger:       logger,
	}
}

func (s *Server) Router() http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(s.recoverer)
	router.Use(s.accessLog)
	router.Use(middleware.Timeout(15 * time.Second))

	router.Get("/healthz", s.health)
	router.Route("/api/v1", func(api chi.Router) {
		api.Post("/auth/login", s.login)
		api.Group(func(protected chi.Router) {
			protected.Use(s.authenticate)
			protected.Get("/auth/me", s.me)
			protected.Post("/auth/logout", s.logout)
			protected.Get("/dashboard", s.dashboard)
			protected.Get("/books", s.listBooks)
			protected.Get("/books/{id}", s.getBook)
			protected.Get("/books/categories", s.categories)
			protected.Post("/books", s.createBook)
			protected.Put("/books/{id}", s.updateBook)
			protected.Delete("/books/{id}", s.deleteBook)
			protected.Get("/members", s.listMembers)
			protected.Post("/members", s.createMember)
			protected.Put("/members/{id}", s.updateMember)
			protected.Delete("/members/{id}", s.deleteMember)
			protected.Get("/loans", s.listLoans)
			protected.Post("/loans", s.createLoan)
			protected.Post("/loans/{id}/return", s.returnLoan)
		})
	})
	return router
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeData(w, r, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_INPUT", "请求格式不正确")
		return
	}
	input.Username = strings.TrimSpace(input.Username)
	if input.Username == "" || input.Password == "" {
		writeError(w, r, http.StatusBadRequest, "INVALID_INPUT", "请输入用户名和密码")
		return
	}
	user, err := s.store.Authenticate(r.Context(), input.Username, input.Password)
	if err != nil {
		if !errors.Is(err, library.ErrNotFound) {
			s.logger.Error("authenticate admin", "error", err, "request_id", middleware.GetReqID(r.Context()))
		}
		writeError(w, r, http.StatusUnauthorized, "INVALID_CREDENTIALS", "用户名或密码错误")
		return
	}
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   user.ID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(12 * time.Hour)),
		Issuer:    "library-management",
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   12 * 60 * 60,
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
	writeData(w, r, http.StatusOK, user)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
	writeData(w, r, http.StatusOK, map[string]bool{"loggedOut": true})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userKey{}).(library.User)
	writeData(w, r, http.StatusOK, user)
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.Dashboard(r.Context())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, data)
}

func (s *Server) listBooks(w http.ResponseWriter, r *http.Request) {
	page, pageSize := pagination(r)
	data, err := s.store.ListBooks(r.Context(), page, pageSize,
		r.URL.Query().Get("search"), r.URL.Query().Get("status"), r.URL.Query().Get("category"))
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, data)
}

func (s *Server) getBook(w http.ResponseWriter, r *http.Request) {
	book, err := s.store.BookByID(
		r.Context(),
		chi.URLParam(r, "id"),
	)
	if err != nil {
		s.businessError(w, r, err)
		return
	}

	writeData(w, r, http.StatusOK, book)
}

func (s *Server) categories(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.Categories(r.Context())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, data)
}

func (s *Server) createBook(w http.ResponseWriter, r *http.Request) {
	var input library.BookInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_INPUT", "请求格式不正确")
		return
	}
	if err := library.ValidateBook(&input); err != nil {
		s.businessError(w, r, err)
		return
	}
	book, err := s.store.CreateBook(r.Context(), currentUserID(r), input)
	if err != nil {
		s.businessError(w, r, err)
		return
	}
	writeData(w, r, http.StatusCreated, book)
}

func (s *Server) updateBook(w http.ResponseWriter, r *http.Request) {
	var input library.BookInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_INPUT", "请求格式不正确")
		return
	}
	if err := library.ValidateBook(&input); err != nil {
		s.businessError(w, r, err)
		return
	}
	book, err := s.store.UpdateBook(r.Context(), currentUserID(r), chi.URLParam(r, "id"), input)
	if err != nil {
		s.businessError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, book)
}

func (s *Server) deleteBook(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteBook(r.Context(), currentUserID(r), chi.URLParam(r, "id")); err != nil {
		s.businessError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, map[string]bool{"deleted": true})
}

func (s *Server) listMembers(w http.ResponseWriter, r *http.Request) {
	page, pageSize := pagination(r)
	data, err := s.store.ListMembers(r.Context(), page, pageSize,
		r.URL.Query().Get("search"), r.URL.Query().Get("status"))
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, data)
}

func (s *Server) createMember(w http.ResponseWriter, r *http.Request) {
	var input library.MemberInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_INPUT", "请求格式不正确")
		return
	}
	if err := library.ValidateMember(&input); err != nil {
		s.businessError(w, r, err)
		return
	}
	member, err := s.store.CreateMember(r.Context(), currentUserID(r), input)
	if err != nil {
		s.businessError(w, r, err)
		return
	}
	writeData(w, r, http.StatusCreated, member)
}

func (s *Server) updateMember(w http.ResponseWriter, r *http.Request) {
	var input library.MemberInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_INPUT", "请求格式不正确")
		return
	}
	if err := library.ValidateMember(&input); err != nil {
		s.businessError(w, r, err)
		return
	}
	member, err := s.store.UpdateMember(r.Context(), currentUserID(r), chi.URLParam(r, "id"), input)
	if err != nil {
		s.businessError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, member)
}

func (s *Server) deleteMember(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteMember(r.Context(), currentUserID(r), chi.URLParam(r, "id")); err != nil {
		s.businessError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, map[string]bool{"deleted": true})
}

func (s *Server) listLoans(w http.ResponseWriter, r *http.Request) {
	page, pageSize := pagination(r)
	data, err := s.store.ListLoans(r.Context(), page, pageSize,
		r.URL.Query().Get("search"), r.URL.Query().Get("status"))
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, data)
}

func (s *Server) createLoan(w http.ResponseWriter, r *http.Request) {
	var input library.LoanInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_INPUT", "请求格式不正确")
		return
	}
	if err := library.ValidateLoan(&input); err != nil {
		s.businessError(w, r, err)
		return
	}
	loan, err := s.store.CreateLoan(r.Context(), currentUserID(r), input)
	if err != nil {
		s.businessError(w, r, err)
		return
	}
	writeData(w, r, http.StatusCreated, loan)
}

func (s *Server) returnLoan(w http.ResponseWriter, r *http.Request) {
	loan, err := s.store.ReturnLoan(r.Context(), currentUserID(r), chi.URLParam(r, "id"))
	if err != nil {
		s.businessError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, loan)
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookie)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "请先登录")
			return
		}
		token, err := jwt.ParseWithClaims(cookie.Value, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return s.jwtSecret, nil
		}, jwt.WithIssuer("library-management"))
		if err != nil || !token.Valid {
			writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "登录状态已过期")
			return
		}
		claims, ok := token.Claims.(*jwt.RegisteredClaims)
		if !ok || claims.Subject == "" {
			writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "登录状态无效")
			return
		}
		user, err := s.store.UserByID(r.Context(), claims.Subject)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "登录状态无效")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, user)))
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.Error("panic recovered", "panic", recovered, "request_id", middleware.GetReqID(r.Context()))
				writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "服务暂时不可用")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(wrapped, r)
		s.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", middleware.GetReqID(r.Context()),
		)
	})
}

func (s *Server) businessError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, library.ErrInvalid):
		writeError(w, r, http.StatusBadRequest, "INVALID_INPUT", cleanMessage(err))
	case errors.Is(err, library.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "NOT_FOUND", "记录不存在或已被删除")
	case errors.Is(err, library.ErrConflict):
		writeError(w, r, http.StatusConflict, "DUPLICATE_RECORD", "ISBN 或编号已存在")
	case errors.Is(err, library.ErrBookUnavailable):
		writeError(w, r, http.StatusConflict, "BOOK_UNAVAILABLE", "该图书当前无可借库存")
	case errors.Is(err, library.ErrMemberInactive):
		writeError(w, r, http.StatusConflict, "MEMBER_INACTIVE", "停用读者不能借阅图书")
	case errors.Is(err, library.ErrLoanLimit):
		writeError(w, r, http.StatusConflict, "LOAN_LIMIT", "该读者已达到同时借阅 5 本的上限")
	case errors.Is(err, library.ErrHasLoans):
		writeError(w, r, http.StatusConflict, "HAS_LOANS", "存在借阅记录，不能删除；可改为停用")
	default:
		s.internalError(w, r, err)
	}
}

func (s *Server) internalError(w http.ResponseWriter, r *http.Request, err error) {
	s.logger.Error("request failed", "error", err, "request_id", middleware.GetReqID(r.Context()))
	writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "服务暂时不可用")
}

func pagination(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}

func currentUserID(r *http.Request) string {
	return r.Context().Value(userKey{}).(library.User).ID
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func cleanMessage(err error) string {
	message := err.Error()
	if index := strings.Index(message, ": "); index >= 0 {
		return message[index+2:]
	}
	return message
}

func writeData(w http.ResponseWriter, r *http.Request, status int, data any) {
	writeJSON(w, status, map[string]any{
		"data":      data,
		"requestId": middleware.GetReqID(r.Context()),
	})
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
		"requestId": middleware.GetReqID(r.Context()),
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
