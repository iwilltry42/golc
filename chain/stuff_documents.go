package chain

import (
	"context"
	"fmt"
	"strings"

	"github.com/hupe1980/golc"
	"github.com/hupe1980/golc/schema"
	"github.com/hupe1980/golc/util"
)

// Compile time check to ensure StuffDocuments satisfies the Chain interface.
var _ schema.Chain = (*StuffDocuments)(nil)

type StuffDocumentsOptions struct {
	*schema.CallbackOptions
	InputKey             string
	DocumentVariableName string
	Separator            string
}

type StuffDocuments struct {
	llmChain *LLM
	opts     StuffDocumentsOptions
}

func NewStuffDocuments(llmChain *LLM, optFns ...func(o *StuffDocumentsOptions)) (*StuffDocuments, error) {
	opts := StuffDocumentsOptions{
		InputKey:             "inputDocuments",
		DocumentVariableName: "context",
		Separator:            "\n\n",
		CallbackOptions: &schema.CallbackOptions{
			Verbose: golc.Verbose,
		},
	}

	for _, fn := range optFns {
		fn(&opts)
	}

	return &StuffDocuments{
		llmChain: llmChain,
		opts:     opts,
	}, nil
}

// Call executes the ConversationalRetrieval chain with the given context and inputs.
// It returns the outputs of the chain or an error, if any.
func (c *StuffDocuments) Call(ctx context.Context, values schema.ChainValues, optFns ...func(o *schema.CallOptions)) (schema.ChainValues, error) {
	opts := schema.CallOptions{}

	for _, fn := range optFns {
		fn(&opts)
	}

	input, ok := values[c.opts.InputKey]
	if !ok {
		return nil, fmt.Errorf("%w: no value for inputKey %s", ErrInvalidInputValues, c.opts.InputKey)
	}

	docs, ok := input.([]schema.Document)
	if !ok {
		return nil, ErrInputValuesWrongType
	}

	contents := []string{}
	for _, doc := range docs {
		contents = append(contents, doc.PageContent)
	}

	inputValues := util.CopyMap(values)
	inputValues[c.opts.DocumentVariableName] = strings.Join(contents, c.opts.Separator)

	return golc.Call(ctx, c.llmChain, inputValues)
}

// Memory returns the memory associated with the chain.
func (c *StuffDocuments) Memory() schema.Memory {
	return nil
}

// Type returns the type of the chain.
func (c *StuffDocuments) Type() string {
	return "StuffDocuments"
}

// Verbose returns the verbosity setting of the chain.
func (c *StuffDocuments) Verbose() bool {
	return c.opts.CallbackOptions.Verbose
}

// Callbacks returns the callbacks associated with the chain.
func (c *StuffDocuments) Callbacks() []schema.Callback {
	return c.opts.CallbackOptions.Callbacks
}

// InputKeys returns the expected input keys.
func (c *StuffDocuments) InputKeys() []string {
	return []string{c.opts.InputKey}
}

// OutputKeys returns the output keys the chain will return.
func (c *StuffDocuments) OutputKeys() []string {
	return c.llmChain.OutputKeys()
}
