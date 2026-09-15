package model

import "testing"

func TestSecretNameAndRedact(t *testing.T) {
	if !SecretName("API_TOKEN") || !SecretName("db-password") || SecretName("PORT") {
		t.Fatal("secret name detection")
	}
	v := Service{Env: []EnvVar{{Name: "API_TOKEN", Value: "abc"}, {Name: "PORT", Value: "80"}}}
	got := v.Redacted()
	if got.Env[0].Value != "" || !got.Env[0].Secret || got.Env[1].Value != "80" {
		t.Fatalf("redacted: %+v", got.Env)
	}
	if v.Env[0].Value != "abc" {
		t.Fatal("redact mutated original")
	}
}
