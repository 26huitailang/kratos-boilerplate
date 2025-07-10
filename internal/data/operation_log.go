package data

import (
	"context"
	"time"

	"kratos-boilerplate/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type operationLogRepo struct {
	data *Data
	log  *log.Helper
}

// NewOperationLogRepo .
func NewOperationLogRepo(data *Data, logger log.Logger) biz.OperationLogRepo {
	return &operationLogRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *operationLogRepo) CreateLog(ctx context.Context, log *biz.OperationLog) error {
	query := `INSERT INTO operation_logs (user_id, username, operation, target, content, result, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.data.db.ExecContext(ctx, query, log.UserID, log.Username, log.Operation, log.Target, log.Content, log.Result, time.Now())
	return err
}

func (r *operationLogRepo) ListLogs(ctx context.Context, userID int64, startTime, endTime time.Time) ([]*biz.OperationLog, error) {
	query := `SELECT id, user_id, username, operation, target, content, result, created_at FROM operation_logs WHERE user_id = $1 AND created_at BETWEEN $2 AND $3 ORDER BY created_at DESC`
	rows, err := r.data.db.QueryContext(ctx, query, userID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*biz.OperationLog
	for rows.Next() {
		log := &biz.OperationLog{}
		err := rows.Scan(&log.ID, &log.UserID, &log.Username, &log.Operation, &log.Target, &log.Content, &log.Result, &log.CreatedAt)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, nil
}
