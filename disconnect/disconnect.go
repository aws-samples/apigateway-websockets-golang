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

	"com.aws-samples/apigateway.websockets.golang/lib/apigw"
	"com.aws-samples/apigateway.websockets.golang/lib/logger"
	"com.aws-samples/apigateway.websockets.golang/lib/redis"
	"github.com/aws/aws-lambda-go/events"

	"github.com/aws/aws-lambda-go/lambda"
	radix "github.com/mediocregopher/radix/v3"
	"go.uber.org/zap"
)

func main() {
	lambda.Start(handler)
}

// handler receives a synchronous invocation from API Gateway when a new connection has been disconnected from the
// application's API. The connection details are removed in the application's Redis cache which cleans up the connection
// details. This handler is not guaranteed to be called when the WebSocket connection is closed.
func handler(ctx context.Context, req *events.APIGatewayWebsocketProxyRequest) (apigw.Response, error) {
	defer func() {
		_ = logger.Instance.Sync()
	}()

	logger.Instance.Info("websocket disconnect",
		zap.String("requestId", req.RequestContext.RequestID),
		zap.String("connectionId", req.RequestContext.ConnectionID))

	var result string
	err := redis.Client.Do(radix.Cmd(&result, "SREM", "connections", req.RequestContext.ConnectionID))
	if err != nil {
		logger.Instance.Error("failed to delete connection details from cache",
			zap.String("requestId", req.RequestContext.RequestID),
			zap.String("connectionId", req.RequestContext.ConnectionID),
			zap.Error(err))

		return apigw.InternalServerErrorResponse(), err
	}

	logger.Instance.Info("websocket connection deleted from cache",
		zap.String("result", result),
		zap.String("requestId", req.RequestContext.RequestID),
		zap.String("connectionId", req.RequestContext.ConnectionID))

	return apigw.OkResponse(), nil
}
