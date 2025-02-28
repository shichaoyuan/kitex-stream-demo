/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * The MIT License (MIT)
 *
 * Copyright (c) 2015-present Aliaksandr Valialkin, VertaMedia, Kirill Danshin, Erik Dubbelboer, FastHTTP Authors
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 * This file may have been modified by CloudWeGo authors. All CloudWeGo
 * Modifications are Copyright 2022 CloudWeGo Authors.
 */

package main

import (
	"context"
	"io"
	"net/http"

	"chatbot/kitex_gen/chatbot"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/hertz-contrib/sse"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

type ChatReq struct {
	Query string `json:"query"`
}

func Chat(ctx context.Context, c *app.RequestContext) {
	var req ChatReq
	err := c.BindAndValidate(&req)
	if err != nil {
		renderError(c, http.StatusBadRequest, err)
		return
	}

	c.SetStatusCode(http.StatusOK)
	s := sse.NewStream(c)

	var message []llms.MessageContent
	history, found := c.Get("history")
	if found {
		message = history.([]llms.MessageContent)
	}
	message = append(message, llms.TextParts(schema.ChatMessageTypeHuman, req.Query))

	resp, err := llm.GenerateContent(ctx, message, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		return s.Publish(&sse.Event{
			Event: "chunk",
			Data:  chunk,
		})
	}))
	if err != nil {
		hlog.CtxErrorf(ctx, "failed to generate: %s", err)
		return
	}

	if len(resp.Choices) > 0 {
		err = s.Publish(&sse.Event{
			Event: "full",
			Data:  []byte(resp.Choices[0].Content),
		})
		if err != nil {
			hlog.CtxErrorf(ctx, "failed to publish: %s", err)
			return
		}
		c.Set("query", req.Query)
		c.Set("response", resp.Choices[0].Content)
	}
}

func SinglePromptKitex(ctx context.Context, c *app.RequestContext) {
	var req ChatReq
	err := c.BindAndValidate(&req)
	if err != nil {
		renderError(c, http.StatusBadRequest, err)
		return
	}

	c.SetStatusCode(http.StatusOK)
	s := sse.NewStream(c)

	rpcReq := &chatbot.Request{Query: &req.Query}
	stream, err := streamClient.Chat(context.Background(), rpcReq)
	if err != nil {
		panic("failed to call Echo: " + err.Error())
	}
	for {
		rpcResp, err := stream.Recv()
		if err == io.EOF {
			hlog.CtxInfof(ctx, "stream is closed")
			break
		} else if err != nil {
			hlog.CtxErrorf(ctx, "failed to generate: %s", err)
			break
		}

		err = s.Publish(&sse.Event{
			Event: *rpcResp.Event,
			Data:  []byte(*rpcResp.Data),
		})
		if err != nil {
			hlog.CtxErrorf(ctx, "failed to publish: %s", err)
			return
		}
	}
}

func SinglePrompt(ctx context.Context, c *app.RequestContext) {
	var req ChatReq
	err := c.BindAndValidate(&req)
	if err != nil {
		renderError(c, http.StatusBadRequest, err)
		return
	}

	c.SetStatusCode(http.StatusOK)
	s := sse.NewStream(c)

	response, err := llms.GenerateFromSinglePrompt(ctx, llm, req.Query, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		return s.Publish(&sse.Event{
			Event: "chunk",
			Data:  chunk,
		})
	}))
	if err != nil {
		hlog.CtxErrorf(ctx, "failed to generate: %s", err)
		return
	}
	err = s.Publish(&sse.Event{
		Event: "full",
		Data:  []byte(response),
	})
	if err != nil {
		hlog.CtxErrorf(ctx, "failed to publish: %s", err)
		return
	}
}

func renderError(c *app.RequestContext, status int, err error) {
	c.JSON(status, map[string]interface{}{
		"message": err,
	})
}
