package service

import "testing"

func TestAuthValidationHelpers(t *testing.T) {
	if !validEmail("owner@example.test") {
		t.Fatal("valid email rejected")
	}
	if validEmail("not-an-email") {
		t.Fatal("invalid email accepted")
	}
	if !passwordOK("123456789012") || passwordOK("short") || passwordOK(string(make([]byte, 73))) {
		t.Fatal("password length validation is wrong")
	}
	if !validName("Roti Sore") || validName("x") {
		t.Fatal("name validation is wrong")
	}
	hashA, hashB := hashToken("a"), hashToken("b")
	if hashA != hashToken("a") || hashA == hashB {
		t.Fatal("token hash is not deterministic")
	}
	firstID, secondID := newID(), newID()
	if len(firstID) != 32 || firstID == secondID {
		t.Fatal("ID generation is wrong")
	}
}
