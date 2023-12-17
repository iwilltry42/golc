package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hupe1980/golc/callback"
	"github.com/hupe1980/golc/model"
	"github.com/hupe1980/golc/model/chatmodel"
	"github.com/hupe1980/golc/prompt"
	"github.com/hupe1980/golc/schema"
)

func main() {
	cohere, err := chatmodel.NewCohere(os.Getenv("COHERE_API_KEY"), func(o *chatmodel.CohereOptions) {
		o.Callbacks = []schema.Callback{callback.NewStreamWriterHandler()}
		o.Stream = true
	})
	if err != nil {
		log.Fatal(err)
	}

	res, err := model.GeneratePrompt(context.Background(), cohere, prompt.StringPromptValue("Write me a song about sparkling water."))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(res.Generations[0].Text)
}
