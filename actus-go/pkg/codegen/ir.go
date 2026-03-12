package codegen

import (
	"time"

	"github.com/shopspring/decimal"
)

// ContractIR (Intermediate Representation) is the abstract representation
// of an ACTUS contract used for code generation across multiple platforms.
type ContractIR struct {
	// Metadata about the contract
	Metadata IRMetadata

	// Contract specification
	Spec ContractSpec

	// State variables
	StateVariables []StateVariable

	// Events (contract events/logs, not ACTUS events)
	Events []Event

	// Functions to be generated
	Functions []Function

	// Storage mappings and data structures
	Storage []StorageItem
}

// IRMetadata contains metadata about the IR
type IRMetadata struct {
	// ContractName is the smart contract name
	ContractName string

	// ACTUSType is the source ACTUS contract type (PAM, LAM, ANN, etc.)
	ACTUSType string

	// Description of the contract
	Description string

	// Author information
	Author string

	// Version of the contract
	Version string

	// License identifier (e.g., "MIT", "Apache-2.0")
	License string
}

// ContractSpec contains the ACTUS contract specification
type ContractSpec struct {
	// ContractType is the ACTUS contract type
	ContractType string

	// Attributes are the ACTUS contract attributes
	Attributes map[string]interface{}

	// ScheduledEvents are the pre-computed event schedule
	ScheduledEvents []ScheduledEvent
}

// ScheduledEvent represents a pre-computed ACTUS event
type ScheduledEvent struct {
	EventType   string
	ScheduledAt time.Time
	Payoff      decimal.Decimal
	Currency    string
}

// StateVariable represents a contract state variable
type StateVariable struct {
	Name         string
	Type         DataType
	Visibility   Visibility
	Description  string
	InitialValue interface{}
	Immutable    bool
}

// Function represents a contract function
type Function struct {
	Name        string
	Type        FunctionType
	Visibility  Visibility
	Parameters  []Parameter
	Returns     []Parameter
	Description string
	Body        FunctionBody
	Modifiers   []string
}

// FunctionType categorizes function types
type FunctionType string

const (
	FunctionTypeConstructor FunctionType = "constructor"
	FunctionTypeView        FunctionType = "view"
	FunctionTypePure        FunctionType = "pure"
	FunctionTypeModifier    FunctionType = "modifier"
	FunctionTypePayable     FunctionType = "payable"
	FunctionTypeNonPayable  FunctionType = "nonpayable"
)

// FunctionBody contains the function implementation logic
type FunctionBody struct {
	// Operations is a sequence of operations
	Operations []Operation

	// RequireChecks are precondition checks
	RequireChecks []RequireCheck

	// EventEmissions are events to emit
	EventEmissions []string
}

// Operation represents a single operation in a function
type Operation struct {
	Type    OperationType
	Target  string
	Args    []interface{}
	Comment string
}

// OperationType categorizes operations
type OperationType string

const (
	OpTypeAssign      OperationType = "assign"
	OpTypeCompute     OperationType = "compute"
	OpTypeStateUpdate OperationType = "state_update"
	OpTypeCall        OperationType = "call"
	OpTypeReturn      OperationType = "return"
	OpTypeEmit        OperationType = "emit"
	OpTypeTransfer    OperationType = "transfer"
)

// RequireCheck represents a precondition check
type RequireCheck struct {
	Condition string
	Message   string
}

// Parameter represents a function parameter or return value
type Parameter struct {
	Name string
	Type DataType
}

// Event represents a contract event/log
type Event struct {
	Name        string
	Parameters  []EventParameter
	Description string
}

// EventParameter represents an event parameter
type EventParameter struct {
	Name    string
	Type    DataType
	Indexed bool
}

// StorageItem represents a storage structure
type StorageItem struct {
	Name        string
	Type        StorageType
	KeyType     DataType
	ValueType   DataType
	Description string
}

// StorageType categorizes storage structures
type StorageType string

const (
	StorageTypeMapping StorageType = "mapping"
	StorageTypeArray   StorageType = "array"
	StorageTypeStruct  StorageType = "struct"
)

// DataType represents a data type in the target language
type DataType struct {
	// Base type (e.g., "uint256", "address", "string")
	BaseType string

	// IsArray indicates if this is an array type
	IsArray bool

	// IsMapping indicates if this is a mapping type
	IsMapping bool

	// KeyType for mappings
	KeyType *DataType

	// ValueType for mappings and arrays
	ValueType *DataType

	// CustomType for structs/custom types
	CustomType string
}

// Visibility represents access control level
type Visibility string

const (
	VisibilityPublic   Visibility = "public"
	VisibilityPrivate  Visibility = "private"
	VisibilityInternal Visibility = "internal"
	VisibilityExternal Visibility = "external"
)

// NewContractIR creates a new ContractIR instance
func NewContractIR(contractName, actusType string) *ContractIR {
	return &ContractIR{
		Metadata: IRMetadata{
			ContractName: contractName,
			ACTUSType:    actusType,
			Version:      "1.0.0",
			License:      "MIT",
		},
		StateVariables: make([]StateVariable, 0),
		Events:         make([]Event, 0),
		Functions:      make([]Function, 0),
		Storage:        make([]StorageItem, 0),
	}
}

// AddStateVariable adds a state variable to the IR
func (ir *ContractIR) AddStateVariable(v StateVariable) {
	ir.StateVariables = append(ir.StateVariables, v)
}

// AddFunction adds a function to the IR
func (ir *ContractIR) AddFunction(f Function) {
	ir.Functions = append(ir.Functions, f)
}

// AddEvent adds an event to the IR
func (ir *ContractIR) AddEvent(e Event) {
	ir.Events = append(ir.Events, e)
}

// AddStorage adds a storage item to the IR
func (ir *ContractIR) AddStorage(s StorageItem) {
	ir.Storage = append(ir.Storage, s)
}
