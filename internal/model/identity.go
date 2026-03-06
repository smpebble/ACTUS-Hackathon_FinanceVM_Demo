package model

import "time"

// CredentialType represents a vLEI credential type.
type CredentialType string

const (
	CredentialTypeLegalEntity CredentialType = "LegalEntity"
	CredentialTypeOOR         CredentialType = "OOR" // Official Organizational Role
	CredentialTypeECR         CredentialType = "ECR" // Engagement Context Role
)

// CredentialStatus represents the status of a vLEI credential.
type CredentialStatus string

const (
	CredentialStatusActive  CredentialStatus = "active"
	CredentialStatusRevoked CredentialStatus = "revoked"
	CredentialStatusExpired CredentialStatus = "expired"
)

// VLEICredential represents a vLEI verifiable credential (adapted from vlei-go).
type VLEICredential struct {
	ID             string           `json:"id"`
	Type           CredentialType   `json:"type"`
	Issuer         string           `json:"issuer"`
	LEI            string           `json:"lei"`
	LegalName      string           `json:"legalName"`
	Jurisdiction   string           `json:"jurisdiction"`
	OfficialRole   string           `json:"officialRole,omitempty"`
	Status         CredentialStatus `json:"status"`
	IssuedAt       time.Time        `json:"issuedAt"`
}

// VLEIVerificationResult represents the result of a vLEI verification.
type VLEIVerificationResult struct {
	IsValid        bool               `json:"isValid"`
	VerifiedAt     time.Time          `json:"verifiedAt"`
	Issuer         *VLEICredential    `json:"issuer"`
	Operator       *VLEICredential    `json:"operator"`
	PolicyName     string             `json:"policyName"`
	Errors         []string           `json:"errors,omitempty"`
}
