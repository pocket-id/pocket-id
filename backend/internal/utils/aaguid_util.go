package utils

import "fmt"

var AAGUIDMap = map[string]string{
	// Bitwarden
	"d548826e-79b4-db40-a3d8-11116f7e8349": "Bitwarden",

	// Google
	"ea9b8d66-4d01-1d21-3ce4-b6b48cb575d4": "Google Password Manager",
	"a5921718-1094-d74c-93df-5582eea98e5c": "Google Password Manager",
	"8e0bf8a9-3a4a-4b0b-966e-2daa78c05f19": "Google Password Manager",

	// Windows Hello
	"6028b017-b1d4-4c02-b4b3-afcdafc96bb2": "Windows Hello",
	"9ddd1817-af5a-4672-a2b9-3e3dd95000a9": "Windows Hello",
	"08987058-cadc-4b81-b6e1-30de50dcbe96": "Windows Hello",
	"6e96969e-a5cf-440d-acea-c73edf8372ef": "Windows Hello",
	"73019266-b7c3-46dd-ba3f-3c1f769a8af3": "Windows Hello",
	"f8a011f3-8c0a-4d15-8006-17111f9edc7d": "Windows Hello",
	"c1f9a0bc-1dd2-404a-b27f-8e29047a43fd": "Windows Hello",
	"53414d53-554e-4700-0000-000000000000": "Windows Hello",
	"4cb9c8df-743a-4767-a70f-a0be4375ee88": "Windows Hello",

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

	// YubiKey
	"dd4ec289-e01d-41c9-bb89-70fa845d4bf2": "YubiKey",
	"ee041b16-0296-4cd7-9a47-83c29b1a0d48": "YubiKey",
	"2fc0579f-8113-47ea-b116-bb5a8db9202d": "YubiKey",
	"cb69481e-8ff7-4039-93ec-0a2729a154a8": "YubiKey",
	"5dcd53ef-5e17-4cd3-8296-37d48ce35f57": "YubiKey",
	"fa2b99dc-9e39-4257-8f92-4a30d23c4118": "YubiKey",
	"ee882879-721c-4913-9775-3dfcce97072a": "YubiKey",
	"c1f9a0bc-1dd2-404a-b27f-8e29047a43f3": "YubiKey",

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

	// iOS/iPadOS
	"0100000000001082": "iOS/iPadOS",
}

// FormatAAGUID converts an AAGUID byte slice to UUID string format
func FormatAAGUID(aaguid []byte) string {
	if len(aaguid) != 16 {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		aaguid[0:4], aaguid[4:6], aaguid[6:8], aaguid[8:10], aaguid[10:16])
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
