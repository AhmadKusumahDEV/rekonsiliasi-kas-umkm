package database

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/database"

const maxQueryTextLength = 1024

// queryTracer connects pgx query execution to the trace provider configured by
// observability.Setup. Arguments are deliberately not recorded because they can
// contain personal data.
type queryTracer struct {
	tracer trace.Tracer
}

var _ pgx.QueryTracer = (*queryTracer)(nil)

func newQueryTracer() pgx.QueryTracer {
	return &queryTracer{tracer: otel.Tracer(instrumentationName)}
}

func (t *queryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx, _ = t.tracer.Start(ctx, "db.query",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system.name", "postgresql"),
			attribute.String("db.operation.name", sqlOperation(data.SQL)),
			attribute.String("db.query.text", truncateQuery(data.SQL)),
			attribute.Int("db.query.parameter_count", len(data.Args)),
		),
	)

	return ctx
}

func (t *queryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	span := trace.SpanFromContext(ctx)
	if data.Err != nil {
		recordDatabaseError(span, data.Err)
	} else if rows := data.CommandTag.RowsAffected(); rows > 0 {
		span.SetAttributes(attribute.Int64("db.response.returned_rows", rows))
	}
	span.End()
}

func recordDatabaseError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func sqlOperation(query string) string {
	fields := strings.Fields(stripLeadingComments(query))
	if len(fields) == 0 {
		return "_OTHER"
	}

	switch operation := strings.ToUpper(fields[0]); operation {
	case "SELECT", "INSERT", "UPDATE", "DELETE", "WITH", "CALL", "DO", "COPY":
		return operation
	default:
		return "_OTHER"
	}
}

func truncateQuery(query string) string {
	query = strings.Join(strings.Fields(stripLeadingComments(query)), " ")
	if len(query) <= maxQueryTextLength {
		return query
	}

	return query[:maxQueryTextLength]
}

// sqlc prefixes generated SQL with a "-- name: QueryName :one" comment. Strip
// only leading comments so the operation and query text stay useful.
func stripLeadingComments(query string) string {
	query = strings.TrimSpace(query)
	for query != "" {
		switch {
		case strings.HasPrefix(query, "--"):
			index := strings.IndexByte(query, '\n')
			if index < 0 {
				return ""
			}
			query = strings.TrimSpace(query[index+1:])
		case strings.HasPrefix(query, "/*"):
			index := strings.Index(query, "*/")
			if index < 0 {
				return ""
			}
			query = strings.TrimSpace(query[index+2:])
		default:
			return query
		}
	}

	return ""
}
