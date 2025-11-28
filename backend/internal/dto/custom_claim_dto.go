package dto

type CustomClaimDto struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	IsLdap bool   `json:"is_ldap"`
}

type CustomClaimCreateDto struct {
	Key    string `json:"key" binding:"required" unorm:"nfc"`
	Value  string `json:"value" binding:"required" unorm:"nfc"`
	IsLdap bool   `json:"is_ldap" unorm:"nfc"`
}

type LdapExtraAttributeDto struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Multi bool
}

type LdapAttributesDto struct {
	User  []LdapExtraAttributeDto `json:"user"`
	Group []LdapExtraAttributeDto `json:"group"`
}
