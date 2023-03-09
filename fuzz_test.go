package main

import (
	"testing"
)

func TestFuzzCrashers(t *testing.T) {

	var crashers = []string{
		"|C2984619140625:",
		"|C9478759765625:",
		"|C :590791705756156:",
		"|C298461940625:",
	}

	for _, f := range crashers {
		decoder := NewPhpDecoder(f)
		_, err := decoder.Decode()
		if err != nil {
			continue
		}
	}
}
