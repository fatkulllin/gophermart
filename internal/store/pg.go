package pg

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/fatkulllin/gophermart/internal/model"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type Store struct {
	conn *sql.DB
}

func NewStore(dsn string) (*Store, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("db ping failed: %w", err)
	}

	return &Store{conn: db}, nil
}

func (s *Store) Bootstrap(fs embed.FS) error {
	goose.SetBaseFS(fs)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("error set dialect postgres %w", err)
	}

	if err := goose.Up(s.conn, "."); err != nil {
		return fmt.Errorf("error run migrate %w", err)
	}
	return nil
}

func (s *Store) SaveUser(ctx context.Context, user model.UserCredentials) (int, error) {
	row := s.conn.QueryRowContext(ctx, "SELECT login FROM users WHERE login = $1", user.Login)
	var userScan string
	err := row.Scan(&userScan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			tx, err := s.conn.BeginTx(ctx, nil)
			if err != nil {
				return 0, fmt.Errorf("pg failed start transaction: %w", err)
			}
			defer tx.Rollback()

			var id int

			row = tx.QueryRowContext(ctx, "INSERT INTO users (login, password_hash) VALUES ($1, $2) RETURNING id", user.Login, user.Password)

			err = row.Scan(&id)

			if err != nil {
				return 0, fmt.Errorf("pg failed to insert new user: %w", err)
			}

			err = tx.Commit()

			if err != nil {
				return 0, fmt.Errorf("pg failed to commit transaction: %w", err)
			}
			return id, nil
		}
		return 0, fmt.Errorf("check existing user: %w", err)
	}
	return 0, model.ErrUserExists
}

func (s *Store) GetUser(ctx context.Context, user model.UserCredentials) (model.User, error) {
	var foundUser model.User
	row := s.conn.QueryRowContext(ctx, "SELECT * FROM users WHERE login = $1", user.Login)
	err := row.Scan(&foundUser.ID, &foundUser.Login, &foundUser.PasswordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, fmt.Errorf("user not found")
		}
		return model.User{}, err
	}
	return foundUser, nil
}
