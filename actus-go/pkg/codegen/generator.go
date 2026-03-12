// Package codegen provides smart contract code generation functionality
// for ACTUS financial contracts across multiple blockchain platforms.
package codegen

import (
	"context"
	"io"
)

// TargetLanguage represents the target blockchain platform/language
type TargetLanguage string

const (
	// LanguageEthereum targets Ethereum blockchain (Solidity)
	LanguageEthereum TargetLanguage = "ethereum"
	// LanguageSolana targets Solana blockchain (Rust/Anchor)
	LanguageSolana TargetLanguage = "solana"
	// LanguageAlgorand targets Algorand blockchain (PyTeal)
	LanguageAlgorand TargetLanguage = "algorand"
	// LanguageCardano targets Cardano blockchain (Plutus/Haskell)
	LanguageCardano TargetLanguage = "cardano"
	// LanguagePolkadot targets Polkadot parachains (ink!/Rust)
	LanguagePolkadot TargetLanguage = "polkadot"
	// LanguageBSC targets Binance Smart Chain (Solidity/BEP-20)
	LanguageBSC TargetLanguage = "binance-smart-chain"
	// LanguageMove targets Aptos/Sui blockchains (Move)
	LanguageMove TargetLanguage = "move"
	// LanguageRustNEAR targets NEAR Protocol (Rust)
	LanguageRustNEAR TargetLanguage = "rust-near"

	// Legacy aliases for backwards compatibility
	LanguageSolidity   = LanguageEthereum
	LanguageRustSolana = LanguageSolana
)

// GeneratorOptions configures code generation behavior
type GeneratorOptions struct {
	// Target blockchain platform/language
	Target TargetLanguage

	// ContractName for the generated smart contract
	ContractName string

	// Optimize enables optimization flags
	Optimize bool

	// IncludeTests generates test files
	IncludeTests bool

	// IncludeDocumentation generates documentation
	IncludeDocumentation bool

	// Version specifies the target language version (e.g., "0.8.20" for Solidity)
	Version string

	// CustomTemplateDir allows custom template directory
	CustomTemplateDir string
}

// GeneratedCode represents the output of code generation
type GeneratedCode struct {
	// Language is the target language
	Language TargetLanguage

	// MainFile contains the primary smart contract code
	MainFile CodeFile

	// AdditionalFiles contains supporting files (interfaces, libraries, tests)
	AdditionalFiles []CodeFile

	// Metadata contains generation metadata
	Metadata GenerationMetadata
}

// CodeFile represents a single generated code file
type CodeFile struct {
	// Filename is the suggested filename
	Filename string

	// Content is the file contents
	Content string

	// FileType indicates the file type (contract, interface, test, etc.)
	FileType FileType
}

// FileType categorizes generated files
type FileType string

const (
	FileTypeContract      FileType = "contract"
	FileTypeInterface     FileType = "interface"
	FileTypeLibrary       FileType = "library"
	FileTypeTest          FileType = "test"
	FileTypeDeployment    FileType = "deployment"
	FileTypeDoc           FileType = "documentation"
	FileTypeConfiguration FileType = "configuration"
)

// GenerationMetadata contains information about the generation process
type GenerationMetadata struct {
	// GeneratedAt timestamp
	GeneratedAt string

	// GeneratorVersion is the codegen module version
	GeneratorVersion string

	// SourceContractType is the ACTUS contract type
	SourceContractType string

	// Warnings contains any generation warnings
	Warnings []string

	// Dependencies lists required dependencies
	Dependencies []Dependency
}

// Dependency represents an external dependency
type Dependency struct {
	Name    string
	Version string
	Source  string
}

// Generator is the interface that all language generators must implement
type Generator interface {
	// Generate produces smart contract code from a ContractIR
	Generate(ctx context.Context, ir *ContractIR, opts GeneratorOptions) (*GeneratedCode, error)

	// Validate checks if the IR can be generated for this target
	Validate(ir *ContractIR) error

	// GetSupportedFeatures returns the ACTUS features supported by this generator
	GetSupportedFeatures() []string

	// GetLanguage returns the target language
	GetLanguage() TargetLanguage
}

// TemplateRenderer handles template-based code generation
type TemplateRenderer interface {
	// Render executes a template with the given data
	Render(templateName string, data interface{}) (string, error)

	// RenderToWriter executes a template and writes to an io.Writer
	RenderToWriter(w io.Writer, templateName string, data interface{}) error

	// LoadTemplates loads templates from a directory
	LoadTemplates(dir string) error
}

// GeneratorFactory creates generators for specific languages
type GeneratorFactory interface {
	// CreateGenerator creates a generator for the specified language
	CreateGenerator(lang TargetLanguage) (Generator, error)

	// ListSupportedLanguages returns all supported languages
	ListSupportedLanguages() []TargetLanguage

	// IsSupported checks if a language is supported
	IsSupported(lang TargetLanguage) bool
}
