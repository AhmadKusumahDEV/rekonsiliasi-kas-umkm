package service

import (
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/repository/rekonsiliasi_repository_sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

func mapEntry(row sqlc.UmkmEntry) models.Entry {
	return models.Entry{
		ID:        row.ID,
		SourceID:  row.SourceID,
		BatchID:   row.BatchID,
		Kind:      row.Kind,
		Date:      dateString(row.OccurredOn),
		Reference: row.Reference,
		Name:      row.Name,
		Amount:    row.Amount,
		Channel:   row.Channel,
		Note:      row.Note,
		Canceled:  row.CanceledAt.Valid,
		CreatedAt: timeValue(row.CreatedAt),
	}
}

func mapImport(row sqlc.UmkmImport) models.Import {
	return models.Import{
		ID:        row.ID,
		Kind:      row.Kind,
		Checksum:  row.Checksum,
		Rows:      int(row.RowCount),
		CreatedAt: timeValue(row.CreatedAt),
		Canceled:  row.CanceledAt.Valid,
	}
}

func mapAllocation(row sqlc.UmkmAllocation) models.Allocation {
	return models.Allocation{
		ID:        row.ID,
		SaleID:    row.SaleEntryID,
		ReceiptID: row.ReceiptEntryID,
		Amount:    row.Amount,
		Fee:       row.Fee,
		Reason:    row.Reason,
		CreatedAt: timeValue(row.CreatedAt),
		Reversed:  row.ReversedAt.Valid,
	}
}

func mapUser(row sqlc.ListUsersByWorkspaceRow) models.User {
	return models.User{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		App:         row.App,
		Name:        row.Name,
		Email:       row.Email,
		Role:        row.Role,
	}
}

func mapAuditEvent(row sqlc.ListAuditEventsByWorkspaceRow) models.AuditEvent {
	return models.AuditEvent{
		ID:        row.ID,
		Actor:     row.Actor,
		Action:    row.Action,
		Detail:    row.Detail,
		CreatedAt: timeValue(row.CreatedAt),
	}
}

func dateString(value pgtype.Date) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format("2006-01-02")
}

func timeValue(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}
