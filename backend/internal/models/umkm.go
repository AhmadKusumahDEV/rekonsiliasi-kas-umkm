package models

import "time"

type Entry struct {
	ID        string    `json:"id"`
	SourceID  string    `json:"source_id"`
	BatchID   string    `json:"batch_id"`
	Kind      string    `json:"kind"`
	Date      string    `json:"date"`
	Reference string    `json:"reference"`
	Name      string    `json:"name"`
	Amount    int64     `json:"amount"`
	Channel   string    `json:"channel"`
	Note      string    `json:"note"`
	Canceled  bool      `json:"canceled"`
	CreatedAt time.Time `json:"created_at"`
}

type Import struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Checksum  string    `json:"checksum"`
	Rows      int       `json:"rows"`
	CreatedAt time.Time `json:"created_at"`
	Canceled  bool      `json:"canceled"`
}

type Allocation struct {
	ID        string    `json:"id"`
	SaleID    string    `json:"sale_id"`
	ReceiptID string    `json:"receipt_id"`
	Amount    int64     `json:"amount"`
	Fee       int64     `json:"fee"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
	Reversed  bool      `json:"reversed"`
}

type Match struct {
	Sale    Entry  `json:"sale"`
	Receipt Entry  `json:"receipt"`
	Reason  string `json:"reason"`
}

type Workspace struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Code    string `json:"code,omitempty"`
	Version int64  `json:"version"`
}

type AuditEvent struct {
	ID        int64     `json:"id"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}

type UMKMState struct {
	Entries     []Entry      `json:"entries"`
	Imports     []Import     `json:"imports"`
	Allocations []Allocation `json:"allocations"`
}

type StateResponse struct {
	Workspace Workspace    `json:"workspace"`
	State     UMKMState    `json:"state"`
	Users     []User       `json:"users"`
	Events    []AuditEvent `json:"events"`
}

type ImportRequest struct {
	Kind string `json:"kind" validate:"required,oneof=sale receipt"`
	CSV  string `json:"csv" validate:"required,max=1048576"`
}

type PreviewResponse struct {
	Rows     []Entry `json:"rows"`
	Count    int     `json:"count"`
	Total    int64   `json:"total"`
	Checksum string  `json:"checksum"`
}

type CommitImportResponse struct {
	ID        string `json:"id"`
	Imported  int    `json:"imported"`
	Skipped   int    `json:"skipped"`
	Duplicate bool   `json:"duplicate,omitempty"`
}

type AllocationRequest struct {
	SaleID    string `json:"sale_id" validate:"required,len=32,hexadecimal"`
	ReceiptID string `json:"receipt_id" validate:"required,len=32,hexadecimal"`
	Amount    int64  `json:"amount" validate:"required,gt=0"`
	Fee       int64  `json:"fee" validate:"gte=0"`
	Reason    string `json:"reason" validate:"max=400"`
	Key       string `json:"key" validate:"required,len=32,hexadecimal"`
}
