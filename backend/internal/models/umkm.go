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
	Category  string    `json:"category,omitempty"`
	Settled   bool      `json:"settled,omitempty"`
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
	ID          int64     `json:"id"`
	Actor       string    `json:"actor"`
	Action      string    `json:"action"`
	Detail      string    `json:"detail"`
	EntityType  *string   `json:"entity_type"`
	EntityID    *string   `json:"entity_id"`
	ActorUserID *string   `json:"actor_user_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// Limits are the server-side import and workspace bounds. The frontend should
// render these values instead of hardcoding its own copy.
type Limits struct {
	MaxImportRows int `json:"max_import_rows"`
	MaxCSVBytes   int `json:"max_csv_bytes"`
	MaxEntries    int `json:"max_entries"`
}

// Exception is one item that still needs a human decision: an unmatched
// entry, an ambiguous candidate, or an entry with a remaining balance.
type Exception struct {
	Entry                 Entry    `json:"entry"`
	Status                string   `json:"status"`
	Remaining             int64    `json:"remaining"`
	CandidateIDs          []string `json:"candidate_ids"`
	CandidateCount        int      `json:"candidate_count"`
	CandidateIDsTruncated bool     `json:"candidate_ids_truncated"`
	SuggestionAvailable   bool     `json:"suggestion_available"`
	Reason                string   `json:"reason"`
	FeeHint               *FeeHint `json:"fee_hint,omitempty"`
}

// FeeHint explains a shortfall between a sale and a matching receipt. It is a
// suggestion only: the user still confirms the allocation and the fee.
type FeeHint struct {
	Amount     int64  `json:"amount"`
	Source     string `json:"source"`
	Label      string `json:"label"`
	Confidence string `json:"confidence"`
	Reason     string `json:"reason"`
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
	CSV  string `json:"csv" validate:"required"`
}

type PreviewResponse struct {
	Rows     []Entry `json:"rows"`
	Count    int     `json:"count"`
	Total    int64   `json:"total"`
	Checksum string  `json:"checksum"`
	Skipped  int     `json:"skipped"`
}

type FlexibleImportRequest struct {
	Kind string `json:"kind" validate:"required,oneof=sale receipt"`
	CSV  string `json:"csv" validate:"required"`
}

type FlexibleImportResponse struct {
	CSV      string   `json:"csv"`
	Count    int      `json:"count"`
	Total    int64    `json:"total"`
	Checksum string   `json:"checksum"`
	Preview  []Entry  `json:"preview"`
	Skipped  int      `json:"skipped"`
	Warnings []string `json:"warnings"`
}

type CommitImportResponse struct {
	ID          string `json:"id"`
	Imported    int    `json:"imported"`
	Skipped     int    `json:"skipped"`
	AutoMatched int    `json:"auto_matched"`
	AIMatched   int    `json:"ai_matched"`
	AIStatus    string `json:"ai_status,omitempty"`
	Duplicate   bool   `json:"duplicate,omitempty"`
}

type AllocationRequest struct {
	SaleID    string `json:"sale_id" validate:"required,len=32,hexadecimal"`
	ReceiptID string `json:"receipt_id" validate:"required,len=32,hexadecimal"`
	Amount    int64  `json:"amount" validate:"required,gt=0"`
	Fee       int64  `json:"fee" validate:"gte=0"`
	Reason    string `json:"reason" validate:"max=400"`
	Key       string `json:"key" validate:"required,len=32,hexadecimal"`
}

type ClearWorkspaceDataRequest struct {
	Password string `json:"password" validate:"required,max=72"`
	Confirm  string `json:"confirm" validate:"required"`
}
