package auth

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/timemachine-auto/timemachine/internal/apperror"
	"github.com/timemachine-auto/timemachine/internal/config"
)

type User struct {
	ID           int64  `json:"id"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	Name         string `json:"name"`
	IsActive     bool   `json:"is_active"`
	passwordHash string
}

type Repo struct{ db *pgxpool.Pool }

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

func (r *Repo) FindByEmail(ctx context.Context, email string) (*User, error) {
	u := &User{}
	err := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, role, name, is_active FROM users
		WHERE lower(email)=lower($1)`, strings.TrimSpace(email)).
		Scan(&u.ID, &u.Email, &u.passwordHash, &u.Role, &u.Name, &u.IsActive)
	if err == pgx.ErrNoRows {
		return nil, apperror.Unauthorized("неверный email или пароль")
	}
	if err != nil {
		return nil, apperror.Internal("find user", err)
	}
	return u, nil
}

// LoginInput is mirrored in handlers.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (in *LoginInput) Validate() error {
	in.Email = strings.TrimSpace(in.Email)
	if !strings.Contains(in.Email, "@") {
		return apperror.Invalid("некорректный email")
	}
	if len(in.Password) < 4 {
		return apperror.Invalid("пароль слишком короткий")
	}
	return nil
}

// IssueToken validates credentials and returns a signed JWT.
func (r *Repo) IssueToken(ctx context.Context, in *LoginInput, cfg config.JWTConfig) (string, *User, error) {
	if err := in.Validate(); err != nil {
		return "", nil, err
	}
	u, err := r.FindByEmail(ctx, in.Email)
	if err != nil {
		return "", nil, err
	}
	if !u.IsActive {
		return "", nil, apperror.Forbidden("пользователь отключён")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.passwordHash), []byte(in.Password)); err != nil {
		return "", nil, apperror.Unauthorized("неверный email или пароль")
	}
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    cfg.Issuer,
		Subject:   strconv.FormatInt(u.ID, 10),
		ExpiresAt: jwt.NewNumericDate(now.Add(cfg.TTL)),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
	}
	extras := map[string]any{
		"email": u.Email,
		"role":  u.Role,
		"uid":   u.ID, // numeric user id, used by middleware.CtxUserIDInt
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, mergeClaims(claims, extras))
	signed, err := token.SignedString([]byte(cfg.Secret))
	if err != nil {
		return "", nil, apperror.Internal("sign token", err)
	}
	return signed, u, nil
}

// HashPassword is used by seed/migrate tooling.
func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

// mergeClaims combines standard claims and private claims into one map.
func mergeClaims(std jwt.RegisteredClaims, extras map[string]any) jwt.MapClaims {
	out := make(jwt.MapClaims, len(extras)+4)
	for k, v := range extras {
		out[k] = v
	}
	if std.Issuer != "" {
		out["iss"] = std.Issuer
	}
	if std.Subject != "" {
		out["sub"] = std.Subject
	}
	if std.ExpiresAt != nil {
		out["exp"] = std.ExpiresAt.Unix()
	}
	if std.IssuedAt != nil {
		out["iat"] = std.IssuedAt.Unix()
	}
	return out
}
