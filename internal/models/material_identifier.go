package models

import "strings"

// NormalizeMaterialIdentifiers keeps TYPENAME (maQuanLy) as the primary
// identifier while retaining the legacy supplies.ID value when it exists.
// Legacy clients that only send ID remain supported by copying it to the
// primary slot; the legacy slot itself is allowed to stay empty.
func NormalizeMaterialIdentifiers(typeName, legacyID string) (string, string) {
	typeName = strings.TrimSpace(typeName)
	legacyID = strings.TrimSpace(legacyID)
	if typeName == "" {
		typeName = legacyID
	}

	return typeName, legacyID
}

// PreferredMaterialCode returns TYPENAME first and falls back to supplies.ID.
func PreferredMaterialCode(typeName, legacyID string) string {
	typeName, legacyID = NormalizeMaterialIdentifiers(typeName, legacyID)
	if typeName != "" {
		return typeName
	}
	return legacyID
}

// MaterialIdentifierKey preserves both values while both generations are in
// use. Once ID disappears, TYPENAME alone remains a stable key.
func MaterialIdentifierKey(typeName, legacyID string) string {
	typeName, legacyID = NormalizeMaterialIdentifiers(typeName, legacyID)
	if typeName != "" && legacyID != "" && !strings.EqualFold(typeName, legacyID) {
		return strings.ToLower(typeName) + "::" + strings.ToLower(legacyID)
	}
	if typeName != "" {
		return strings.ToLower(typeName)
	}
	return strings.ToLower(legacyID)
}
