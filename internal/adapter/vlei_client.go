package adapter

import (
	"time"

	"github.com/google/uuid"

	"github.com/smpebble/actus-fvm/internal/model"
)

// VLEIClient provides vLEI identity verification (mock mode, no GLEIF connection).
type VLEIClient struct{}

// NewVLEIClient creates a new vLEI client.
func NewVLEIClient() *VLEIClient {
	return &VLEIClient{}
}

// CreateIssuerCredential creates a mock Legal Entity vLEI credential for the issuer.
func (c *VLEIClient) CreateIssuerCredential(lei, legalName, jurisdiction string) *model.VLEICredential {
	return &model.VLEICredential{
		ID:           uuid.New().String(),
		Type:         model.CredentialTypeLegalEntity,
		Issuer:       "GLEIF Root of Trust (Mock)",
		LEI:          lei,
		LegalName:    legalName,
		Jurisdiction: jurisdiction,
		Status:       model.CredentialStatusActive,
		IssuedAt:     time.Now(),
	}
}

// CreateOperatorCredential creates a mock OOR vLEI credential for the operator.
func (c *VLEIClient) CreateOperatorCredential(lei, personName, role string) *model.VLEICredential {
	return &model.VLEICredential{
		ID:           uuid.New().String(),
		Type:         model.CredentialTypeOOR,
		Issuer:       "Qualified vLEI Issuer (Mock)",
		LEI:          lei,
		LegalName:    personName,
		OfficialRole: role,
		Status:       model.CredentialStatusActive,
		IssuedAt:     time.Now(),
	}
}

// VerifyTransaction verifies both issuer and operator credentials for a transaction.
func (c *VLEIClient) VerifyTransaction(issuer, operator *model.VLEICredential) *model.VLEIVerificationResult {
	result := &model.VLEIVerificationResult{
		IsValid:    true,
		VerifiedAt: time.Now(),
		Issuer:     issuer,
		Operator:   operator,
		PolicyName: "FinanceVM Asset Tokenisation Policy",
	}

	if issuer.Status != model.CredentialStatusActive {
		result.IsValid = false
		result.Errors = append(result.Errors, "Issuer credential is not active")
	}
	if operator.Status != model.CredentialStatusActive {
		result.IsValid = false
		result.Errors = append(result.Errors, "Operator credential is not active")
	}

	return result
}
