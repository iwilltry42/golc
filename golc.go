// Package golc provides functions for executing chains.
package golc

import (
	"context"
	"errors"

	"github.com/hupe1980/golc/callback"
	"github.com/hupe1980/golc/schema"
	"golang.org/x/sync/errgroup"
)

var (
	// Verbose controls the verbosity of the chain execution.
	Verbose = false

	// ErrMultipleInputs is returned when calling a chain with more than one expected input is not supported.
	ErrMultipleInputs = errors.New("chain with more than one expected input")
	// ErrMultipleOutputs is returned when calling a chain with more than one expected output is not supported.
	ErrMultipleOutputs = errors.New("chain with more than one expected output")
	// ErrMultipleOutputs is returned when calling a chain with more than one expected output is not supported.
	ErrWrongOutputType = errors.New("chain with non string return type")
)

type CallOptions struct {
	Callbacks      []schema.Callback
	IncludeRunInfo bool
	Stop           []string
}

// Call executes a chain with multiple inputs.
// It returns the outputs of the chain or an error, if any.
func Call(ctx context.Context, chain schema.Chain, inputs schema.ChainValues, optFns ...func(*CallOptions)) (schema.ChainValues, error) {
	opts := CallOptions{
		IncludeRunInfo: false,
	}

	for _, fn := range optFns {
		fn(&opts)
	}

	cm := callback.NewManager(opts.Callbacks, chain.Callbacks(), chain.Verbose())

	rm, err := cm.OnChainStart(chain.Type(), inputs)
	if err != nil {
		return nil, err
	}

	if chain.Memory() != nil {
		vars, _ := chain.Memory().LoadMemoryVariables(ctx, inputs)
		for k, v := range vars {
			inputs[k] = v
		}
	}

	outputs, err := chain.Call(ctx, inputs, func(o *schema.CallOptions) {
		o.CallbackManger = rm
		o.Stop = opts.Stop
	})
	if err != nil {
		if cbErr := rm.OnChainError(err); cbErr != nil {
			return nil, cbErr
		}

		return nil, err
	}

	if chain.Memory() != nil {
		if err := chain.Memory().SaveContext(ctx, inputs, outputs); err != nil {
			return nil, err
		}
	}

	if err := rm.OnChainEnd(outputs); err != nil {
		return nil, err
	}

	if opts.IncludeRunInfo {
		outputs["runInfo"] = cm.RunID()
	}

	return outputs, nil
}

type SimpleCallOptions struct {
	Callbacks []schema.Callback
	Stop      []string
}

// SimpleCall executes a chain with a single input and a single output.
// It returns the output value as a string or an error, if any.
func SimpleCall(ctx context.Context, chain schema.Chain, input any, optFns ...func(*SimpleCallOptions)) (string, error) {
	opts := SimpleCallOptions{}

	for _, fn := range optFns {
		fn(&opts)
	}

	if len(chain.InputKeys()) != 1 {
		return "", ErrMultipleInputs
	}

	if len(chain.OutputKeys()) != 1 {
		return "", ErrMultipleOutputs
	}

	outputValues, err := Call(ctx, chain, map[string]any{chain.InputKeys()[0]: input}, func(o *CallOptions) {
		o.Callbacks = opts.Callbacks
		o.Stop = opts.Stop
	})
	if err != nil {
		return "", err
	}

	outputValue, ok := outputValues[chain.OutputKeys()[0]].(string)
	if !ok {
		return "", ErrWrongOutputType
	}

	return outputValue, nil
}

type BatchCallOptions struct {
	Callbacks []schema.Callback
	Stop      []string
}

// BatchCall executes multiple calls to the chain.Call function concurrently and collects
// the results in the same order as the inputs. It utilizes the errgroup package to manage
// the concurrent execution and handle any errors that may occur.
func BatchCall(ctx context.Context, chain schema.Chain, inputs []schema.ChainValues, optFns ...func(*BatchCallOptions)) ([]schema.ChainValues, error) {
	opts := BatchCallOptions{}

	for _, fn := range optFns {
		fn(&opts)
	}

	errs, errctx := errgroup.WithContext(ctx)

	chainValues := make([]schema.ChainValues, len(inputs))

	for i, input := range inputs {
		i, input := i, input

		errs.Go(func() error {
			vals, err := Call(errctx, chain, input, func(o *CallOptions) {
				o.Callbacks = opts.Callbacks
				o.Stop = opts.Stop
			})
			if err != nil {
				return err
			}

			chainValues[i] = vals

			return nil
		})
	}

	if err := errs.Wait(); err != nil {
		return nil, err
	}

	return chainValues, nil
}
