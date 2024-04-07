package main

import (
	chatbot "chatbot/kitex_gen/chatbot"
	"context"

	"github.com/tmc/langchaingo/llms"
)

// TestServiceImpl implements the last service interface defined in the IDL.
type TestServiceImpl struct{}

var chunkEvent = "chunk"
var fullEvent = "full"

func (s *TestServiceImpl) Chat(req *chatbot.Request, stream chatbot.TestService_ChatServer) (err error) {
	response, err := llms.GenerateFromSinglePrompt(context.Background(), llm, *req.Query, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		chunkStr := string(chunk)
		return stream.Send(&chatbot.Response{Event: &chunkEvent, Data: &chunkStr})
	}))
	if err != nil {
		return err
	}
	err = stream.Send(&chatbot.Response{Event: &fullEvent, Data: &response})
	if err != nil {
		return err
	}
	return nil
}
