package report

// ReadError is a filesystem failure on a path: it does not exist, or cannot
// be read. It is deliberately not an absence of runs — a broken mount or a
// permission problem reported as "no run here" sends a user to debug the
// wrong thing.
type ReadError struct {
	Path string
	Err  error
}

func (e *ReadError) Error() string {
	return "cannot read " + e.Path + ": " + e.Err.Error()
}

func (e *ReadError) Unwrap() error {
	return e.Err
}

// OpenError is parsec's refusal to read a log, with the path it refused.
// parsec's own message names the version and range, the bytes it found, or
// where the log was cut; this type only adds the path.
type OpenError struct {
	Path string
	Err  error
}

func (e *OpenError) Error() string {
	return e.Path + ": " + e.Err.Error()
}

func (e *OpenError) Unwrap() error {
	return e.Err
}
