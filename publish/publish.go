// MIT No Attribution

// Copyright 2020 Amazon.com, Inc. or its affiliates.

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"time"

	"com.aws-samples/apigateway.websockets.golang/lib/apigw"
	"com.aws-samples/apigateway.websockets.golang/lib/apigw/ws"
	"com.aws-samples/apigateway.websockets.golang/lib/logger"
	"com.aws-samples/apigateway.websockets.golang/lib/redis"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/aws/external"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	radix "github.com/mediocregopher/radix/v3"
	"go.uber.org/zap"
)

// Stack is a simple thread-safe Pop only stack implementation which allows workers to pull work from the stack.
type Stack struct {
	mu       sync.Mutex
	elements []string
}

// Pop pops the next item from the stack and returns it or returns an error if the stack is empty.
func (s *Stack) Pop() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	c := len(s.elements)
	if c == 0 {
		return "", errors.New("no more elements")
	}

	v := s.elements[c-1]
	s.elements = s.elements[:c-1]
	return v, nil
}

// Len returns the length of the stack.
func (s *Stack) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.elements)
}

// cfg is the base or parent AWS configuration for this lambda.
var cfg aws.Config

// apiClient provides access to the Amazon API Gateway management functions. Once initialized, the instance is reused
// across subsequent AWS Lambda invocations. This potentially amortizes the instance creation over multiple executions
// of the AWS Lambda instance.
var apiClient *apigatewaymanagementapi.Client

// Use the SDK default configuration, loading additional config and credentials values from the environment variables,
// shared credentials, and shared configuration files.
func init() {
	var err error
	cfg, err = external.LoadDefaultAWSConfig()
	if err != nil {
		logger.Instance.Panic("unable to load SDK config", zap.Error(err))
	}
}

func main() {
	lambda.Start(handler)
}

// handler is the hook AWS Lambda calls to invoke the function as an Amazon API Gateway Proxy. This handlers reads the
// request and echos the request back out to all connected clients. This demonstrates looking up connected clients from
// the Redis cache and calling the Amazon API Gateway Management API to send data to the connected clients.
func handler(ctx context.Context, req *events.APIGatewayWebsocketProxyRequest) (apigw.Response, error) {
	defer func() {
		_ = logger.Instance.Sync()
	}()

	// Lazily initialize the API Gateway Management client. This enables setting the service's endpoint to our API
	// endpoint. These values are provided from the synchronous request, thus the client can only be created upon the
	// first invocation.
	if apiClient == nil {
		apiClient = apigw.NewAPIGatewayManagementClient(&cfg, req.RequestContext.DomainName, req.RequestContext.Stage)
	}

	logger.Instance.Info("websocket publish",
		zap.String("requestId", req.RequestContext.RequestID),
		zap.String("connectionId", req.RequestContext.ConnectionID))

	input, err := new(ws.InputEnvelop).Decode([]byte(req.Body))
	if err != nil {
		logger.Instance.Error("failed to parse client input",
			zap.String("requestId", req.RequestContext.RequestID),
			zap.String("connectionId", req.RequestContext.ConnectionID),
			zap.Error(err))

		return apigw.BadRequestResponse(), err
	}

	output := &ws.OutputEnvelop{
		Data:     input.Data,
		Type:     input.Type,
		Received: time.Now().Unix(),
	}

	data, err := output.Encode()
	if err != nil {
		logger.Instance.Error("failed to encode output",
			zap.String("requestId", req.RequestContext.RequestID),
			zap.String("connectionId", req.RequestContext.ConnectionID),
			zap.Error(err))

		return apigw.InternalServerErrorResponse(), err
	}

	stack := new(Stack)
	err = redis.Client.Do(radix.Cmd(&(stack.elements), "SMEMBERS", "connections"))
	if err != nil {
		logger.Instance.Error("failed to read connections from cache",
			zap.String("requestId", req.RequestContext.RequestID),
			zap.String("connectionId", req.RequestContext.ConnectionID),
			zap.Error(err))

		return apigw.InternalServerErrorResponse(), err
	}

	logger.Instance.Info("websocket connections read from cache",
		zap.Int("connections", stack.Len()),
		zap.String("requestId", req.RequestContext.RequestID),
		zap.String("connectionId", req.RequestContext.ConnectionID))

	// Calculate how many go routines should be created to handle the work. Taking the number of logical CPUs times a
	// factor of 4 enables processing outgoing messages concurrently while limiting the amount of context switching.
	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU()*4; i++ {
		wg.Add(1)

		// Run the go routine until the context is canceled or there is no more work to process.
		go func(sender string, echo bool) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Pull the next connection id from the stack. If the stack is empty, a non-nil error is returned
					// and the go routine exits cleanly. Otherwise, a nil error and the next connection id is returned.
					id, err := stack.Pop()
					if err != nil {
						return
					}

					// Do not send data to the connection if the connection represents the sender and the message was
					// configured to not echo back the message.
					if id == sender && !echo {
						continue
					}

					// Publish the data to the connected client via Amazon API Gateway's Management API. If publishing
					// the data results in an error, the error is passed to a convenience function which attempts to
					// resolve the issue which caused the error. The convenience function may return the same error if
					// it can not be handled or may return a different error if attempting the resolution results in an
					// error. Regardless, if an error is returned the only course of action is to log it.
					err = handleError(publish(ctx, id, data), id)
					if err != nil {
						logger.Instance.Error("failed to publish to connection",
							zap.String("receiver", id),
							zap.String("requestId", req.RequestContext.RequestID),
							zap.String("sender", req.RequestContext.ConnectionID),
							zap.Error(err))
					}
				}
			}
		}(req.RequestContext.ConnectionID, input.Echo)
	}

	wg.Wait()
	return apigw.OkResponse(), nil
}

// publish publishes the provided data to the provided Amazon API Gateway connection ID. A common failure scenario which
// results in an error is if the connection ID is no longer valid. This can occur when a client disconnected from the
// Amazon API Gateway endpoint but the disconnect AWS Lambda was not invoked as it is not guaranteed to be invoked when
// clients disconnect.
func publish(ctx context.Context, id string, data []byte) error {
	_, err := apiClient.PostToConnectionRequest(&apigatewaymanagementapi.PostToConnectionInput{
		Data:         data,
		ConnectionId: aws.String(id),
	}).Send(ctx)

	return err
}

// handleError is a convenience function for taking action for a given error value. The function handles nil errors as a
// convenience to the caller. If a nil error is provided, the error is immediately returned. The function may return an
// error from the handling action, such as deleting the id from the cache, if that action results in an error.
func handleError(err error, id string) error {
	if err == nil {
		return err
	}

	// Casting to the awserr.Error type will allow you to inspect the error code returned by the service in code. The
	// error code can be used to switch on context specific functionality.
	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case aws.ErrCodeSerialization:
			logger.Instance.Info("delete stale connection details from cache", zap.String("connectionId", id))
			return deleteConnectionId(id)
		case apigatewaymanagementapi.ErrCodeGoneException:
			logger.Instance.Info("delete stale connection details from cache", zap.String("connectionId", id))
			return deleteConnectionId(id)
		default:
			return err
		}
	}

	return err
}

// deleteConnectionId deletes the connection id from the REDIS cache. The function logs both error and success cases.
func deleteConnectionId(id string) error {
	var result string
	err := redis.Client.Do(radix.Cmd(&result, "SREM", "connections", id))
	if err != nil {
		logger.Instance.Error("failed to delete connection details from cache",
			zap.String("connectionId", id),
			zap.Error(err))

		return err
	}

	logger.Instance.Info("websocket connection deleted from cache",
		zap.String("result", result),
		zap.String("connectionId", id))

	return err
}
