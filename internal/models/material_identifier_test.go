package models

import "testing"

func TestNormalizeMaterialIdentifiers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		typeName      string
		legacyID      string
		wantTypeName  string
		wantLegacyID  string
		wantPreferred string
		wantKey       string
	}{
		{
			name:          "typename is preferred while id is retained",
			typeName:      " A33011 ",
			legacyID:      " OLD-33011 ",
			wantTypeName:  "A33011",
			wantLegacyID:  "OLD-33011",
			wantPreferred: "A33011",
			wantKey:       "a33011::old-33011",
		},
		{
			name:          "future row can omit id",
			typeName:      "A33011",
			wantTypeName:  "A33011",
			wantPreferred: "A33011",
			wantKey:       "a33011",
		},
		{
			name:          "legacy client can send id only",
			legacyID:      "OLD-33011",
			wantTypeName:  "OLD-33011",
			wantLegacyID:  "OLD-33011",
			wantPreferred: "OLD-33011",
			wantKey:       "old-33011",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			typeName, legacyID := NormalizeMaterialIdentifiers(test.typeName, test.legacyID)
			if typeName != test.wantTypeName || legacyID != test.wantLegacyID {
				t.Fatalf("NormalizeMaterialIdentifiers() = (%q, %q), want (%q, %q)", typeName, legacyID, test.wantTypeName, test.wantLegacyID)
			}
			if got := PreferredMaterialCode(test.typeName, test.legacyID); got != test.wantPreferred {
				t.Fatalf("PreferredMaterialCode() = %q, want %q", got, test.wantPreferred)
			}
			if got := MaterialIdentifierKey(test.typeName, test.legacyID); got != test.wantKey {
				t.Fatalf("MaterialIdentifierKey() = %q, want %q", got, test.wantKey)
			}
		})
	}
}
