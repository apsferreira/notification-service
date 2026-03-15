package repository

import (
	"context"
	"time"

	"github.com/institutoitinerante/notification-service/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OTPRepository struct {
	db *pgxpool.Pool
}

func NewOTPRepository(db *pgxpool.Pool) *OTPRepository {
	return &OTPRepository{
		db: db,
	}
}

func (r *OTPRepository) Create(ctx context.Context, otp *model.OTPCode) error {
	query := `INSERT INTO otp_codes (id, email, code_hash, channel, attempts, expires_at, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.Exec(ctx, query,
		otp.ID, otp.Email, otp.CodeHash, otp.Channel,
		otp.Attempts, otp.ExpiresAt, otp.CreatedAt,
	)
	return err
}

func (r *OTPRepository) FindLatestByEmail(ctx context.Context, email string) (*model.OTPCode, error) {
	otp := &model.OTPCode{}
	query := `SELECT id, email, code_hash, channel, attempts, expires_at, created_at
	          FROM otp_codes
	          WHERE email = $1 AND expires_at > NOW()
	          ORDER BY created_at DESC LIMIT 1`
	err := r.db.QueryRow(ctx, query, email).Scan(
		&otp.ID, &otp.Email, &otp.CodeHash, &otp.Channel,
		&otp.Attempts, &otp.ExpiresAt, &otp.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return otp, nil
}

func (r *OTPRepository) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE otp_codes SET attempts = attempts + 1 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *OTPRepository) DeleteByEmail(ctx context.Context, email string) error {
	query := `DELETE FROM otp_codes WHERE email = $1`
	_, err := r.db.Exec(ctx, query, email)
	return err
}

func (r *OTPRepository) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM otp_codes WHERE expires_at < NOW()`
	_, err := r.db.Exec(ctx, query)
	return err
}

func (r *OTPRepository) CountRecentByEmail(ctx context.Context, email string, since time.Time) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM otp_codes WHERE email = $1 AND created_at > $2`
	err := r.db.QueryRow(ctx, query, email, since).Scan(&count)
	return count, err
}