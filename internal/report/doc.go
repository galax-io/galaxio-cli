// Package report reads a finished load-test run and summarises it.
//
// It is glue over github.com/galax-io/parsec plus one pass: it finds the run,
// opens its log whatever version wrote it, and walks the stream once, counting
// what the log holds into a Summary. It computes the statistics of the run
// itself, because parsec yields primitives and exports no statistic; it writes
// no data format, and it declares no type for anything parsec already defines —
// the position of a sample, the bounds of a run, the outcome recorded, an
// optional value, and what a source can never record are all parsec's own.
//
// The stats.json writer of galaxio-cli#52 folds over this same pass.
package report
