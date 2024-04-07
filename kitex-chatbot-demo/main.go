package main

import (
	chatbot "chatbot/kitex_gen/chatbot/testservice"
	"log"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

var llm llms.Model

func main() {
	var err error
	llm, err = ollama.New(ollama.WithModel("llama2"))
	if err != nil {
		panic(err)
	}

	svr := chatbot.NewServer(new(TestServiceImpl))

	err = svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
