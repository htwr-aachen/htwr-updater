package main

import "testing"

func TestCreateHMAC(t *testing.T) {
	secret := "It's a Secret to Everybody"
	payload := "Hello, World!"

	result := CreateHMAC(secret, []byte(payload))

	if result != "sha256=757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17" {
		t.Errorf("expected %s != got %s", "sha256=757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17", result)
	}
}

func TestVerifyHMAC(t *testing.T) {
	secret := "It's a Secret to Everybody"
	payload := "Hello, World!"

	result := VerifyHMAC(secret, "sha256=757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17", []byte(payload))

	if !result {
		t.Errorf("Result should be true")
	}
}
