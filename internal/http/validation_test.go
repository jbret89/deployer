package httpapi

import "testing"

func TestIsValidRepoName(t *testing.T) {
	validNames := []string{"app", "app-1", "APP_1", "service_api"}
	for _, name := range validNames {
		if !isValidRepoName(name) {
			t.Fatalf("expected %q to be valid", name)
		}
	}

	invalidNames := []string{"", "../app", "app/name", "app.name", "app\\name", "app name"}
	for _, name := range invalidNames {
		if isValidRepoName(name) {
			t.Fatalf("expected %q to be invalid", name)
		}
	}
}
