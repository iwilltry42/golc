package chain

import (
	"context"
	"strings"

	"github.com/hupe1980/golc"
	"github.com/hupe1980/golc/model"
	"github.com/hupe1980/golc/prompt"
	"github.com/hupe1980/golc/schema"
)

// Compile time check to ensure LLM satisfies the Chain interface.
var _ schema.Chain = (*LLM)(nil)

type LLMOptions struct {
	*schema.CallbackOptions
	Memory       schema.Memory
	OutputKey    string
	OutputParser schema.OutputParser[any]
}

type LLM struct {
	llm    schema.LLM
	prompt *prompt.Template
	opts   LLMOptions
}

func NewLLM(llm schema.LLM, prompt *prompt.Template, optFns ...func(o *LLMOptions)) (*LLM, error) {
	opts := LLMOptions{
		OutputKey: "text",
		CallbackOptions: &schema.CallbackOptions{
			Verbose: golc.Verbose,
		},
	}

	for _, fn := range optFns {
		fn(&opts)
	}

	return &LLM{
		prompt: prompt,
		llm:    llm,
		opts:   opts,
	}, nil
}

// Call executes the ConversationalRetrieval chain with the given context and inputs.
// It returns the outputs of the chain or an error, if any.
func (c *LLM) Call(ctx context.Context, inputs schema.ChainValues, optFns ...func(o *schema.CallOptions)) (schema.ChainValues, error) {
	opts := schema.CallOptions{}

	for _, fn := range optFns {
		fn(&opts)
	}

	promptValue, err := c.prompt.FormatPrompt(inputs)
	if err != nil {
		return nil, err
	}

	res, err := model.GeneratePrompt(ctx, c.llm, []schema.PromptValue{promptValue}, func(o *model.Options) {
		o.Stop = opts.Stop

		if opts.CallbackManger != nil {
			o.Callbacks = opts.CallbackManger.GetInheritableCallbacks()
			o.ParentRunID = opts.CallbackManger.RunID()
		}
	})
	if err != nil {
		return nil, err
	}

	return schema.ChainValues{
		c.opts.OutputKey: c.getFinalOutput(res.Generations),
	}, nil
}

func (c *LLM) Prompt() *prompt.Template {
	return c.prompt
}

// Memory returns the memory associated with the chain.
func (c *LLM) Memory() schema.Memory {
	return c.opts.Memory
}

// Type returns the type of the chain.
func (c *LLM) Type() string {
	return "LLM"
}

// Verbose returns the verbosity setting of the chain.
func (c *LLM) Verbose() bool {
	return c.opts.CallbackOptions.Verbose
}

// Callbacks returns the callbacks associated with the chain.
func (c *LLM) Callbacks() []schema.Callback {
	return c.opts.CallbackOptions.Callbacks
}

// InputKeys returns the expected input keys.
func (c *LLM) InputKeys() []string {
	return c.prompt.InputVariables()
}

// OutputKeys returns the output keys the chain will return.
func (c *LLM) OutputKeys() []string {
	return []string{c.opts.OutputKey}
}

func (c *LLM) getFinalOutput(generations [][]*schema.Generation) string {
	output := []string{}
	for _, generation := range generations {
		// Get the text of the top generated string.
		output = append(output, strings.TrimSpace(generation[0].Text))
	}

	return output[0]
}
