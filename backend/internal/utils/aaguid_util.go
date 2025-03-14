package utils

import (
	"fmt"
	"log"
)

// The map below can be used for manually looking up authenticators that are not in the FIDO MDS.

var AAGUIDMap = map[string]string{
	// Bitwarden
	"d548826e-79b4-db40-a3d8-11116f7e8349": "Bitwarden",

	// 1Password
	"98b62f56-8a06-425b-85dd-49d6ca4eaab2": "1Password",

	// Dashlane
	"1e1ef5e6-cf33-4949-88c0-2ee55b2d8024": "Dashlane",

	// Apple
	"2fc0579f-8113-47ea-b116-bb5a8db9202a": "Apple",
	"c5ef55ff-ad9a-4b9f-b580-adebafe026d0": "Apple",

	// Safari
	"adce0002-35bc-c60a-648b-0b25f1f05503": "Safari",

	// Passkeys.com
	"cf03be56-5b65-4228-9b38-3a023eb81d4a": "Passkeys.com",

	// LastPass
	"7c0903ab-d6c7-46dd-a25c-08d44c756f7c": "LastPass",

	// Chrome
	"a9139ba2-ad26-4367-9e67-3efebeabd979": "Chrome",
	"aba949cc-b70e-486a-87d9-d677b4c71a53": "Chrome",

	// Firefox
	"08987058-cadc-4b81-b6e1-30de50dcbe95": "Firefox",
	"cd443203-676d-45a5-a8a2-a21bd2f69e59": "Firefox",

	// Microsoft Authenticator
	"73bb0cd4-e502-49b8-9c6f-b59445bf720b": "Microsoft Authenticator",

	// Duo
	"ee041b16-0296-4cd7-9a47-83c29b1a0d49": "Duo",

	// Keeper
	"9c7dd906-68fd-4b76-8efd-da19f6c8c5ff": "Keeper",
	"b0458839-dd3a-4458-a516-6c30c84955b6": "Keeper",
	"b93fd961-f2e6-462f-b122-82002247de78": "Keeper",

	// Enpass
	"154b9b58-9a88-466b-851e-7ca3b65169a4": "Enpass",

	// Brave
	"34f5766d-1536-4a24-9033-0e294e510fb0": "Brave",

	// NordPass
	"ec99db3c-d248-4aa4-aff1-3509d7e96fb2": "NordPass",

	// Samsung Pass
	"0133eeee-d6d1-4c1e-a394-2187c052bad7": "Samsung Pass",

	// Proton Pass
	"c5411d09-f3e7-4a01-99f8-16e245fc584d": "Proton Pass",

	// Firefox Lockwise
	"85203421-48f9-4355-9bc8-8a53846e5083": "Firefox Lockwise",

	// Edge
	"9d86b3f0-71db-4de8-a8e0-e2a1013c1c31": "Edge",
}

// FormatAAGUID converts an AAGUID byte slice to UUID string format
func FormatAAGUID(aaguid []byte) string {
	// Log the raw AAGUID bytes for debugging
	log.Printf("Raw AAGUID bytes: %v", aaguid)

	if len(aaguid) == 0 {
		return ""
	}

	// If exactly 16 bytes, format as UUID
	if len(aaguid) == 16 {
		return fmt.Sprintf("%x-%x-%x-%x-%x",
			aaguid[0:4], aaguid[4:6], aaguid[6:8], aaguid[8:10], aaguid[10:16])
	}

	// Otherwise just return as hex
	return fmt.Sprintf("%x", aaguid)
}

// LookupAuthenticatorName looks up an authenticator name from its AAGUID
func LookupAuthenticatorName(aaguid []byte) string {
	aaguidStr := FormatAAGUID(aaguid)
	if aaguidStr == "" {
		return ""
	}

	if name, ok := AAGUIDMap[aaguidStr]; ok {
		return name
	}

	return ""
}
