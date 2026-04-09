package application

import (
	"time"

	"github.com/diptopandit/tork/internal/domain"
)

// FilterBuilder accumulates filter criteria using a fluent API.
type FilterBuilder struct {
	f domain.TaskFilter
}

// NewFilterBuilder returns an empty FilterBuilder.
func NewFilterBuilder() *FilterBuilder {
	return &FilterBuilder{}
}

// WithLists restricts results to the supplied list IDs.
func (b *FilterBuilder) WithLists(ids ...string) *FilterBuilder {
	b.f.ListIDs = append(b.f.ListIDs, ids...)
	return b
}

// WithStatuses restricts results to the supplied statuses.
func (b *FilterBuilder) WithStatuses(ss ...domain.Status) *FilterBuilder {
	b.f.Statuses = append(b.f.Statuses, ss...)
	return b
}

// WithPriorities restricts results to the supplied priorities.
func (b *FilterBuilder) WithPriorities(ps ...domain.Priority) *FilterBuilder {
	b.f.Priorities = append(b.f.Priorities, ps...)
	return b
}

// WithTags restricts results to tasks that carry all listed tags.
func (b *FilterBuilder) WithTags(tags ...string) *FilterBuilder {
	b.f.Tags = append(b.f.Tags, tags...)
	return b
}

// WithSearch sets the full-text search query.
func (b *FilterBuilder) WithSearch(q string) *FilterBuilder {
	b.f.Search = q
	return b
}

// WithDueBefore restricts results to tasks whose due date is before t.
func (b *FilterBuilder) WithDueBefore(t time.Time) *FilterBuilder {
	b.f.DueBefore = &t
	return b
}

// Build returns the constructed TaskFilter.
func (b *FilterBuilder) Build() domain.TaskFilter {
	return b.f
}
