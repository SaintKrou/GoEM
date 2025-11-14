package repository

import (
	"fmt"
	"log"
	"subscription-service/internal/model"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type SubscriptionRepository struct {
	db *sqlx.DB
}

func NewSubscriptionRepository(db *sqlx.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) Create(sub *model.Subscription) error {
	query := `
		INSERT INTO subscriptions (id, service_name, price, user_id, start_date, end_date)
		VALUES (:id, :service_name, :price, :user_id, :start_date, :end_date)
	`

	_, err := r.db.NamedExec(query, sub)
	if err != nil {
		log.Printf("Ошибка при вставке подписки в БД: %v", err)
		return err
	}

	return nil
}

func (r *SubscriptionRepository) GetByID(id uuid.UUID) (*model.Subscription, error) {
	var sub model.Subscription
	query := "SELECT * FROM subscriptions WHERE id = $1"
	err := r.db.Get(&sub, query, id)
	if err != nil {
		log.Printf("Ошибка при получении подписки по ID %s: %v", id, err)
		return nil, err
	}
	return &sub, nil
}

func (r *SubscriptionRepository) List(user_id *uuid.UUID, service_name *string) ([]model.Subscription, error) {
	query := "SELECT * FROM subscriptions WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if user_id != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, *user_id)
		argIndex++
	}

	if service_name != nil {
		query += fmt.Sprintf(" AND service_name = $%d", argIndex)
		args = append(args, *service_name)
	}

	query += " ORDER BY start_date DESC"

	var subs []model.Subscription
	err := r.db.Select(&subs, query, args...)
	if err != nil {
		log.Printf("Ошибка при получении списка подписок: %v", err)
		return nil, err
	}

	return subs, nil
}

func (r *SubscriptionRepository) GetTotalCost(startMonth, endMonth string, userID *uuid.UUID, serviceName *string) (int, error) {
	if !isValidMonthYear(startMonth) || !isValidMonthYear(endMonth) {
		return 0, fmt.Errorf("некорректный формат месяца: должен быть MM-YYYY")
	}

	query := `
		SELECT COALESCE(SUM(price), 0)
		FROM subscriptions
		WHERE (end_date IS NULL OR end_date >= $1)
		  AND start_date <= $2
	`
	args := []interface{}{startMonth, endMonth}
	argIndex := 3

	if userID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, *userID)
		argIndex++
	}

	if serviceName != nil {
		query += fmt.Sprintf(" AND service_name = $%d", argIndex)
		args = append(args, *serviceName)
	}

	var total int
	err := r.db.Get(&total, query, args...)
	if err != nil {
		log.Printf("Ошибка при расчёте общей стоимости: %v", err)
		return 0, err
	}

	return total, nil
}

func isValidMonthYear(s string) bool {
	_, err := time.Parse("01-2006", s)
	return err == nil
}

func (r *SubscriptionRepository) Update(sub *model.Subscription) error {
	query := `
		UPDATE subscriptions
		SET service_name = :service_name,
		    price = :price,
		    user_id = :user_id,
		    start_date = :start_date,
		    end_date = :end_date
		WHERE id = :id
	`

	result, err := r.db.NamedExec(query, sub)
	if err != nil {
		log.Printf("Ошибка при обновлении подписки %s: %v", sub.ID, err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("подписка с ID %s не найдена", sub.ID)
	}

	return nil
}

func (r *SubscriptionRepository) Delete(id uuid.UUID) error {
	query := "DELETE FROM subscriptions WHERE id = $1"
	result, err := r.db.Exec(query, id)
	if err != nil {
		log.Printf("Ошибка при удалении подписки %s: %v", id, err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("подписка с ID %s не найдена", id)
	}

	return nil
}
