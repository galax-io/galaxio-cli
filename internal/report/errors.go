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
