package main

import "testing"

func TestArch(t *testing.T) {
	s, err := arch("ndk-strip.386")
	if err != nil || s != "386" {
		t.Fatalf("faild!!! %q", err)
	}

	s, err = arch("ndk-strip.amd64")
	if err != nil || s != "amd64" {
		t.Fatalf("faild!!! %q", err)
	}
}
