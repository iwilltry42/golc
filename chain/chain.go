package chain

import (
	"context"
	"strings"

	"github.com/hupe1980/golc/schema"
)

func Run(ctx context.Context, chain schema.Chain, input any) (string, error) {
	inputKeys := chain.InputKeys()
	if len(inputKeys) != 1 {
		return "", ErrMultipleInputsInRun
	}

	outputKeys := chain.OutputKeys()
	if len(outputKeys) != 1 {
		return "", ErrMultipleOutputsInRun
	}

	inputValues := map[string]any{inputKeys[0]: input}

	outputValues, err := Call(ctx, chain, inputValues)
	if err != nil {
		return "", err
	}

	outputValue, ok := outputValues[outputKeys[0]].(string)
	if !ok {
		return "", ErrWrongOutputTypeInRun
	}

	return strings.TrimSpace(outputValue), nil
}

func Call(ctx context.Context, chain schema.Chain, inputs schema.ChainValues) (schema.ChainValues, error) {
	return chain.Call(ctx, inputs)
}

func Apply(ctx context.Context, chain schema.Chain, inputs []schema.ChainValues) ([]schema.ChainValues, error) {
	chainValues := []schema.ChainValues{}

	for _, input := range inputs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			vals, err := chain.Call(ctx, input)
			if err != nil {
				return nil, err
			}

			chainValues = append(chainValues, vals)
		}
	}

	return chainValues, nil
}

type callbackOptions struct {
	Callbacks []schema.Callback
	Verbose   bool
}
