package chain

import (
	"context"
	"fmt"
	"strings"

	"github.com/hupe1980/golc"
	"github.com/hupe1980/golc/prompt"
	"github.com/hupe1980/golc/schema"
	"github.com/hupe1980/golc/util"
)

// Compile time check to ensure RefineDocuments satisfies the Chain interface.
var _ schema.Chain = (*RefineDocuments)(nil)

type RefineDocumentsOptions struct {
	*schema.CallbackOptions
	InputKey             string
	DocumentVariableName string
	InitialResponseName  string
	DocumentPrompt       *prompt.Template
	OutputKey            string
}

type RefineDocuments struct {
	llmChain       *LLM
	refineLLMChain *LLM
	opts           RefineDocumentsOptions
}

func NewRefineDocuments(llmChain *LLM, refineLLMChain *LLM, optFns ...func(o *RefineDocumentsOptions)) (*RefineDocuments, error) {
	opts := RefineDocumentsOptions{
		InputKey:             "inputDocuments",
		DocumentVariableName: "context",
		InitialResponseName:  "existingAnswer",
		OutputKey:            "text",
		CallbackOptions: &schema.CallbackOptions{
			Verbose: golc.Verbose,
		},
	}

	for _, fn := range optFns {
		fn(&opts)
	}

	if opts.DocumentPrompt == nil {
		p, err := prompt.NewTemplate("{{.pageContent}}")
		if err != nil {
			return nil, err
		}

		opts.DocumentPrompt = p
	}

	return &RefineDocuments{
		llmChain:       llmChain,
		refineLLMChain: refineLLMChain,
		opts:           opts,
	}, nil
}

// Call executes the ConversationalRetrieval chain with the given context and inputs.
// It returns the outputs of the chain or an error, if any.
func (c *RefineDocuments) Call(ctx context.Context, values schema.ChainValues, optFns ...func(o *schema.CallOptions)) (schema.ChainValues, error) {
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

	if len(docs) == 0 {
		return nil, fmt.Errorf("%w: documents slice has no elements", ErrInvalidInputValues)
	}

	rest := util.OmitByKeys(values, []string{c.opts.InputKey})

	initialInputs, err := c.constructInitialInputs(docs[0], rest)
	if err != nil {
		return nil, err
	}

	res, err := golc.SimpleCall(ctx, c.llmChain, initialInputs)
	if err != nil {
		return nil, err
	}

	for i := 1; i < len(docs); i++ {
		refineInputs, err := c.constructRefineInputs(docs[i], res, rest)
		if err != nil {
			return nil, err
		}

		res, err = golc.SimpleCall(ctx, c.refineLLMChain, refineInputs)
		if err != nil {
			return nil, err
		}
	}

	return map[string]any{
		c.opts.OutputKey: strings.TrimSpace(res),
	}, nil
}

// Memory returns the memory associated with the chain.
func (c *RefineDocuments) Memory() schema.Memory {
	return nil
}

// Type returns the type of the chain.
func (c *RefineDocuments) Type() string {
	return "RefineDocuments"
}

// Verbose returns the verbosity setting of the chain.
func (c *RefineDocuments) Verbose() bool {
	return c.opts.CallbackOptions.Verbose
}

// Callbacks returns the callbacks associated with the chain.
func (c *RefineDocuments) Callbacks() []schema.Callback {
	return c.opts.CallbackOptions.Callbacks
}

// InputKeys returns the expected input keys.
func (c *RefineDocuments) InputKeys() []string {
	return []string{c.opts.InputKey}
}

// OutputKeys returns the output keys the chain will return.
func (c *RefineDocuments) OutputKeys() []string {
	return c.llmChain.OutputKeys()
}

func (c *RefineDocuments) constructInitialInputs(doc schema.Document, rest map[string]any) (map[string]any, error) {
	docInfo := make(map[string]any)

	docInfo["pageContent"] = doc.PageContent
	for key, value := range doc.Metadata {
		docInfo[key] = value
	}

	inputs := util.CopyMap(rest)

	docText, err := c.opts.DocumentPrompt.Format(docInfo)
	if err != nil {
		return nil, err
	}

	inputs[c.opts.DocumentVariableName] = docText

	return inputs, nil
}

func (c *RefineDocuments) constructRefineInputs(doc schema.Document, lastResponse string, rest map[string]any) (map[string]any, error) {
	docInfo := make(map[string]any)

	docInfo["pageContent"] = doc.PageContent

	for key, value := range doc.Metadata {
		docInfo[key] = value
	}

	inputs := util.CopyMap(rest)

	docText, err := c.opts.DocumentPrompt.Format(docInfo)
	if err != nil {
		return nil, err
	}

	inputs[c.opts.DocumentVariableName] = docText
	inputs[c.opts.InitialResponseName] = lastResponse

	return inputs, nil
}
