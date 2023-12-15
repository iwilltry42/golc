package chatmodel

import (
	"context"
	"fmt"

	"github.com/hupe1980/golc"
	"github.com/hupe1980/golc/callback"
	"github.com/hupe1980/golc/integration/ollama"
	"github.com/hupe1980/golc/schema"
	"github.com/hupe1980/golc/tokenizer"
	"github.com/hupe1980/golc/util"
)

// Compile time check to ensure Ollama satisfies the ChatModel interface.
var _ schema.ChatModel = (*Ollama)(nil)

// OllamaClient is an interface for the Ollama generative model client.
type OllamaClient interface {
	// GenerateChat produces a single request and response for the Ollama generative model.
	GenerateChat(ctx context.Context, req *ollama.ChatRequest) (*ollama.ChatResponse, error)
}

// OllamaOptions contains options for the Ollama model.
type OllamaOptions struct {
	// CallbackOptions specify options for handling callbacks during text generation.
	*schema.CallbackOptions `map:"-"`
	// Tokenizer represents the tokenizer to be used with the LLM model.
	schema.Tokenizer `map:"-"`
	// ModelName is the name of the Gemini model to use.
	ModelName string `map:"model_name,omitempty"`
	// Temperature controls the randomness of the generation. Higher values make the output more random.
	Temperature float32 `map:"temperature,omitempty"`
	// MaxTokens is the maximum number of tokens to generate in the completion.
	MaxTokens int `map:"max_tokens,omitempty"`
	// TopP is the nucleus sampling parameter. It controls the cumulative probability of the most likely tokens to sample from.
	TopP float32 `map:"top_p,omitempty"`
	// TopK is the number of top tokens to consider for sampling.
	TopK int `map:"top_k,omitempty"`
	// PresencePenalty penalizes repeated tokens.
	PresencePenalty float32 `map:"presence_penalty,omitempty"`
	// FrequencyPenalty penalizes repeated tokens according to frequency.
	FrequencyPenalty float32 `map:"frequency_penalty,omitempty"`
}

// Ollama is a struct representing the Ollama generative model.
type Ollama struct {
	schema.Tokenizer
	client OllamaClient
	opts   OllamaOptions
}

// NewOllama creates a new instance of the Ollama model with the provided client and options.
func NewOllama(client OllamaClient, optFns ...func(o *OllamaOptions)) (*Ollama, error) {
	opts := OllamaOptions{
		CallbackOptions: &schema.CallbackOptions{
			Verbose: golc.Verbose,
		},
		ModelName:        "llama2",
		Temperature:      0.7,
		MaxTokens:        256,
		TopP:             1,
		PresencePenalty:  0,
		FrequencyPenalty: 0,
	}

	for _, fn := range optFns {
		fn(&opts)
	}

	if opts.Tokenizer == nil {
		var tErr error

		opts.Tokenizer, tErr = tokenizer.NewGPT2()
		if tErr != nil {
			return nil, tErr
		}
	}

	return &Ollama{
		Tokenizer: opts.Tokenizer,
		client:    client,
		opts:      opts,
	}, nil
}

// Generate generates text based on the provided chat messages and options.
func (cm *Ollama) Generate(ctx context.Context, messages schema.ChatMessages, optFns ...func(o *schema.GenerateOptions)) (*schema.ModelResult, error) {
	opts := schema.GenerateOptions{
		CallbackManger: &callback.NoopManager{},
	}

	for _, fn := range optFns {
		fn(&opts)
	}

	ollamaMessages := make([]ollama.Message, len(messages))

	for i, m := range messages {
		switch m.Type() { // nolint exhaustive
		case schema.ChatMessageTypeSystem:
			ollamaMessages[i] = ollama.Message{Role: "system", Content: m.Content()}
		case schema.ChatMessageTypeAI:
			ollamaMessages[i] = ollama.Message{Role: "assistant", Content: m.Content()}
		case schema.ChatMessageTypeHuman:
			ollamaMessages[i] = ollama.Message{Role: "user", Content: m.Content()}
		default:
			return nil, fmt.Errorf("unknown message type: %s", m.Type())
		}
	}

	res, err := cm.client.GenerateChat(ctx, &ollama.ChatRequest{
		Model:    cm.opts.ModelName,
		Messages: ollamaMessages,
		Stream:   util.AddrOrNil(false),
		Options: ollama.Options{
			Temperature:      cm.opts.Temperature,
			NumPredict:       cm.opts.MaxTokens,
			TopK:             cm.opts.TopK,
			TopP:             cm.opts.TopP,
			PresencePenalty:  cm.opts.PresencePenalty,
			FrequencyPenalty: cm.opts.FrequencyPenalty,
			Stop:             opts.Stop,
		},
	})
	if err != nil {
		return nil, err
	}

	return &schema.ModelResult{
		Generations: []schema.Generation{newChatGeneraton(res.Message.Content)},
		LLMOutput:   map[string]any{},
	}, nil
}

// Type returns the type of the model.
func (cm *Ollama) Type() string {
	return "chatmodel.Ollama"
}

// Verbose returns the verbosity setting of the model.
func (cm *Ollama) Verbose() bool {
	return cm.opts.Verbose
}

// Callbacks returns the registered callbacks of the model.
func (cm *Ollama) Callbacks() []schema.Callback {
	return cm.opts.Callbacks
}

// InvocationParams returns the parameters used in the model invocation.
func (cm *Ollama) InvocationParams() map[string]any {
	return util.StructToMap(cm.opts)
}
