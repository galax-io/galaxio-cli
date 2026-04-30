package main

import (
	"encoding/json"
	"fmt"
	"io"
)

const (
	outputText = "text"
	outputJSON = "json"
)

func validateOutputFormat(format string) error {
	switch format {
	case outputText, outputJSON:
		return nil
	default:
		return UsageError{Err: fmt.Errorf("unsupported output format %q", format)}
	}
}

func writeJSON(writer io.Writer, value any) error {
	payload, err := encodeJSON(value)
	if err != nil {
		return err
	}
	_, err = writer.Write(payload)
	return err
}

func encodeJSON(value any) ([]byte, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return append(payload, '\n'), nil
}
