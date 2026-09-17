// Package report reads a finished load-test run.
//
// It is glue over github.com/galax-io/parsec plus one counting pass: it finds
// the run, opens its log whatever version wrote it, and walks the stream once
// to count what the log holds. It computes no statistic and writes no data
// format, and it declares no type for anything parsec already defines — the
// position of a sample, the bounds of a run, the outcome recorded, an optional
// value, and what a source can never record are all parsec's own.
//
// The statistics of galaxio-cli#51 and the stats.json writer of galaxio-cli#52
// fold over this same pass.
package report
