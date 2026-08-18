package postgres

import (
	"database/sql"
	"encoding/json"
	"time"

	"ispilolite/internal/models"
	"ispilolite/pkg/database"
	"ispilolite/pkg/notifications"
)

type notificationRepository struct{ dbReader, dbWriter *sql.DB }

func NewNotificationRepository() *notificationRepository {
	return &notificationRepository{dbReader: database.GetReader(), dbWriter: database.GetWriter()}
}

func (r *notificationRepository) CreateNotification(notification *models.Notification) error {
	data, err := json.Marshal(notification.Data)
	if err != nil {
		return err
	}
	_, err = r.dbWriter.Exec(`INSERT INTO notifications (id,user_id,type,title,message,data,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (user_id,type,(data->>'location_id')) WHERE type='coverage_recommendation' DO UPDATE SET title=EXCLUDED.title,message=EXCLUDED.message,data=EXCLUDED.data`, notification.ID, notification.UserID, notification.Type, notification.Title, notification.Message, data, notification.CreatedAt)
	if err == nil {
		notifications.Default.Publish(notification)
	}
	return err
}

func (r *notificationRepository) ListNotifications(userID string, unreadOnly bool, limit int) ([]*models.Notification, error) {
	rows, err := r.dbReader.Query(`SELECT id,user_id,type,title,message,data,read_at,created_at FROM notifications WHERE user_id=$1 AND (NOT $2 OR read_at IS NULL) ORDER BY created_at DESC LIMIT $3`, userID, unreadOnly, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*models.Notification, 0)
	for rows.Next() {
		item := &models.Notification{}
		var data []byte
		var readAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.UserID, &item.Type, &item.Title, &item.Message, &data, &readAt, &item.CreatedAt); err != nil {
			return nil, err
		}
		if readAt.Valid {
			value := readAt.Time
			item.ReadAt = &value
		}
		if err := json.Unmarshal(data, &item.Data); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *notificationRepository) MarkNotificationRead(userID, notificationID string) error {
	result, err := r.dbWriter.Exec(`UPDATE notifications SET read_at=$1 WHERE id=$2 AND user_id=$3`, time.Now().UTC(), notificationID, userID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
