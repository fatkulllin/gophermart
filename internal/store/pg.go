package pg

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/fatkulllin/gophermart/internal/logger"
	"github.com/fatkulllin/gophermart/internal/model"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
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

func (s *Store) ExistUser(ctx context.Context, user model.UserCredentials) (bool, error) {
	row := s.conn.QueryRowContext(ctx, "SELECT login FROM users WHERE login = $1", user.Login)
	var userScan string
	err := row.Scan(&userScan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("check existing user: %w", err)
	}
	return true, nil
}

func (s *Store) CreateUser(ctx context.Context, user model.UserCredentials) (int, error) {

	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("pg failed start transaction: %w", err)
	}
	defer tx.Rollback()

	var id int

	row := tx.QueryRowContext(ctx, "INSERT INTO users (login, password_hash) VALUES ($1, $2) RETURNING id", user.Login, user.Password)

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

func (s *Store) SaveOrder(ctx context.Context, user model.User, orderNumber int64) (model.Order, int64, error) {
	row := s.conn.QueryRowContext(ctx, "SELECT user_id, order_number, status FROM orders WHERE order_number = $1", orderNumber)
	var founderOrder model.Order
	err := row.Scan(&founderOrder.UserID, &founderOrder.OrderNumber, &founderOrder.Status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			tx, err := s.conn.BeginTx(ctx, nil)
			if err != nil {
				return founderOrder, 0, fmt.Errorf("pg failed start transaction: %w", err)
			}
			defer tx.Rollback()

			resultInsert, err := tx.ExecContext(ctx, "INSERT INTO orders (user_id, order_number, status) VALUES ($1, $2, $3);", user.ID, orderNumber, "NEW")

			if err != nil {
				return founderOrder, 0, fmt.Errorf("pg failed to insert new order %v login %s: %w", orderNumber, user.Login, err)
			}

			err = tx.Commit()

			if err != nil {
				return founderOrder, 0, fmt.Errorf("pg failed to commit transaction: %w", err)
			}
			rowsAffect, _ := resultInsert.RowsAffected()
			return model.Order{UserID: user.ID, OrderNumber: orderNumber, Status: "NEW"}, rowsAffect, nil
		}
		return model.Order{}, 0, fmt.Errorf("check existing order_number: %w", err)
	}
	return founderOrder, 0, nil
}

func (s *Store) GetOrders() ([]model.Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	listOrderNumbersScan := make([]model.Order, 0)

	rows, err := s.conn.QueryContext(ctx, "SELECT order_number, status FROM orders WHERE status IN ('NEW', 'PROCESSING');")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var order model.Order
		err = rows.Scan(&order.OrderNumber, &order.Status)
		if err != nil {
			return nil, err
		}

		listOrderNumbersScan = append(listOrderNumbersScan, order)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	logger.Log.Debug("get orders new and PROCESSING", zap.Any("order", listOrderNumbersScan))
	return listOrderNumbersScan, nil
}

func (s *Store) UpdateOrderStatus(ctx context.Context, order model.AccrualOrderResponse) error {
	logger.Log.Debug("UpdateOrderStatus order", zap.Any("order", order))
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, "UPDATE orders SET status = $2, accrual = $3 WHERE order_number = $1;")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, order.Order, order.Status, order.Accrual)
	if err != nil {
		return fmt.Errorf("storage failed to update status %s order %d error: %s", order.Status, order.Order, err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("storage failed to commit transaction: %v error: %s", order.Order, err)
	}

	return nil
}

func (s *Store) GetUserBalance(ctx context.Context, userID int) (float64, float64, error) {
	row := s.conn.QueryRowContext(ctx, "SELECT (SELECT COALESCE(SUM(accrual), 0) FROM orders WHERE user_id = $1 AND status = 'PROCESSED'), (SELECT COALESCE(SUM(amount), 0) FROM withdrawals WHERE user_id = $1);", userID)
	var accrual, withdrawn float64
	err := row.Scan(&accrual, &withdrawn)
	if err != nil {
		return 0, 0, err
	}
	return accrual, withdrawn, nil
}

func (s *Store) GetUserOrders(ctx context.Context, userID int) ([]model.Order, error) {
	listOrders := make([]model.Order, 0)
	rows, err := s.conn.QueryContext(ctx, "SELECT order_number, status, accrual, uploaded_at FROM orders WHERE user_id = $1 ORDER BY uploaded_at DESC", userID)
	if err != nil {
		return []model.Order{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var order model.Order
		err = rows.Scan(&order.OrderNumber, &order.Status, &order.Accrual, &order.UploadedAt)
		if err != nil {
			return []model.Order{}, err
		}

		listOrders = append(listOrders, order)
	}
	err = rows.Err()
	if err != nil {
		return []model.Order{}, err
	}
	return listOrders, nil
}

func (s *Store) InsertWithdrawal(ctx context.Context, withdraw model.Withdrawal) error {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO withdrawals (user_id, order_number, amount) VALUES ($1, $2, $3);")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, withdraw.UserID, withdraw.OrderNumber, withdraw.Amount)
	if err != nil {
		return fmt.Errorf("storage failed to add withdraw user_id %d order_number %d error: %w", withdraw.UserID, withdraw.OrderNumber, err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("storage failed to commit transaction:  user_id %d order_number %d error: %w", withdraw.UserID, withdraw.OrderNumber, err)
	}

	return nil
}

func (s *Store) GetWithdrawals(ctx context.Context, userID int) ([]model.Withdrawal, error) {
	listWithdrawal := make([]model.Withdrawal, 0)
	rows, err := s.conn.QueryContext(ctx, "SELECT order_number, amount, processed_at FROM withdrawals WHERE user_id = $1 ORDER BY processed_at DESC", userID)
	if err != nil {
		return []model.Withdrawal{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var withdrawal model.Withdrawal
		err = rows.Scan(&withdrawal.OrderNumber, &withdrawal.Amount, &withdrawal.ProcessedAt)
		if err != nil {
			return []model.Withdrawal{}, err
		}

		listWithdrawal = append(listWithdrawal, withdrawal)
	}
	err = rows.Err()
	if err != nil {
		return []model.Withdrawal{}, err
	}
	return listWithdrawal, nil
}
