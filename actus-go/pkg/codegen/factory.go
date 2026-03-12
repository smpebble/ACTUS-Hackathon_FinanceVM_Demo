package codegen

import (
	"fmt"
)

// DefaultGeneratorFactory is the default implementation of GeneratorFactory
type DefaultGeneratorFactory struct {
	generators map[TargetLanguage]Generator
}

// NewGeneratorFactory creates a new GeneratorFactory instance
func NewGeneratorFactory() GeneratorFactory {
	factory := &DefaultGeneratorFactory{
		generators: make(map[TargetLanguage]Generator),
	}

	// Note: Import cycles prevented, generators will be registered externally
	// Usage: factory.RegisterGenerator(LanguageSolidity, solidity.NewGenerator())

	return factory
}

// RegisterGenerator registers a generator for a specific language
func (f *DefaultGeneratorFactory) RegisterGenerator(lang TargetLanguage, gen Generator) {
	f.generators[lang] = gen
}

// CreateGenerator creates a generator for the specified language
func (f *DefaultGeneratorFactory) CreateGenerator(lang TargetLanguage) (Generator, error) {
	gen, exists := f.generators[lang]
	if !exists {
		return nil, fmt.Errorf("unsupported target language: %s", lang)
	}
	return gen, nil
}

// ListSupportedLanguages returns all supported languages
func (f *DefaultGeneratorFactory) ListSupportedLanguages() []TargetLanguage {
	languages := make([]TargetLanguage, 0, len(f.generators))
	for lang := range f.generators {
		languages = append(languages, lang)
	}
	return languages
}

// IsSupported checks if a language is supported
func (f *DefaultGeneratorFactory) IsSupported(lang TargetLanguage) bool {
	_, exists := f.generators[lang]
	return exists
}
