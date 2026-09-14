// Package report turns a finished load-test run into the records that
// galaxio report writes.
//
// It converts the canonical stream parsec yields into the documented record
// schema and encodes it as JSON Lines without retaining a record. It computes
// nothing: no count, mean, percentile, range or series. The only numbers it
// produces are the per-kind tallies in Summary, which a caller prints as a
// diagnostic and never as a statistic.
package report
