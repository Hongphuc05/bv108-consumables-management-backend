package models

import "testing"

func TestSelectCompanyContactByNameMatches(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input []CompanyContact
		check func(t *testing.T, got *CompanyContact)
	}{
		{
			name:  "returns nil when there are no matches",
			input: nil,
			check: func(t *testing.T, got *CompanyContact) {
				t.Helper()
				if got != nil {
					t.Fatalf("expected nil, got %+v", got)
				}
			},
		},
		{
			name: "keeps unique match unchanged",
			input: []CompanyContact{
				{
					MaSoThue:  "123",
					TenCongTy: "Cong ty A",
					ID:        "123",
					TaxID:     "123",
					Email:     "a@example.com",
					Gmail:     "a@example.com",
				},
			},
			check: func(t *testing.T, got *CompanyContact) {
				t.Helper()
				if got == nil {
					t.Fatal("expected contact, got nil")
				}
				if got.MaSoThue != "123" || got.Email != "a@example.com" || got.ID != "123" {
					t.Fatalf("unexpected contact: %+v", got)
				}
			},
		},
		{
			name: "drops tax id when duplicate names map to different tax ids but same email",
			input: []CompanyContact{
				{
					MaSoThue:    "105168916",
					TenCongTy:   "Cong ty B",
					ID:          "105168916",
					TaxID:       "105168916",
					Email:       "same@example.com",
					Gmail:       "same@example.com",
					IdentityKey: buildCompanyIdentityKey("Cong ty B", "105168916"),
				},
				{
					MaSoThue:    "301856443",
					TenCongTy:   "Cong ty B",
					ID:          "301856443",
					TaxID:       "301856443",
					Email:       "same@example.com",
					Gmail:       "same@example.com",
					IdentityKey: buildCompanyIdentityKey("Cong ty B", "301856443"),
				},
			},
			check: func(t *testing.T, got *CompanyContact) {
				t.Helper()
				if got == nil {
					t.Fatal("expected contact, got nil")
				}
				if got.MaSoThue != "" || got.ID != "" || got.TaxID != "" {
					t.Fatalf("expected tax id fields to be cleared, got %+v", got)
				}
				if got.Email != "same@example.com" || got.Gmail != "same@example.com" {
					t.Fatalf("expected shared email to be preserved, got %+v", got)
				}
			},
		},
		{
			name: "drops both tax id and email when duplicate names disagree on email",
			input: []CompanyContact{
				{
					MaSoThue:    "111",
					TenCongTy:   "Cong ty C",
					ID:          "111",
					TaxID:       "111",
					Email:       "first@example.com",
					Gmail:       "first@example.com",
					IdentityKey: buildCompanyIdentityKey("Cong ty C", "111"),
				},
				{
					MaSoThue:    "222",
					TenCongTy:   "Cong ty C",
					ID:          "222",
					TaxID:       "222",
					Email:       "second@example.com",
					Gmail:       "second@example.com",
					IdentityKey: buildCompanyIdentityKey("Cong ty C", "222"),
				},
			},
			check: func(t *testing.T, got *CompanyContact) {
				t.Helper()
				if got == nil {
					t.Fatal("expected contact, got nil")
				}
				if got.MaSoThue != "" || got.ID != "" || got.TaxID != "" {
					t.Fatalf("expected tax id fields to be cleared, got %+v", got)
				}
				if got.Email != "" || got.Gmail != "" {
					t.Fatalf("expected ambiguous email to be cleared, got %+v", got)
				}
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.check(t, selectCompanyContactByNameMatches(tc.input))
		})
	}
}
