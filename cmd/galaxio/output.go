package main

import (
	"encoding/json"
	"fmt"
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

func encodeJSON(value interface{}) ([]byte, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return append(payload, '\n'), nil
}
