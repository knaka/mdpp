package main

import (
	"os"
	"testing"
)

func TestMain(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"help option", []string{"--help"}, false},
		{"invalid option", []string{"--foo"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			err := mdppMain(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("args=%v, wantErr=%v, got err=%v", tt.args, tt.wantErr, err)
			}
		})
	}
}
