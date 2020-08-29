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

// Package redis provides a singleton Redis client instance which is used across the same AWS Lambda execution contexts.
// Reusing the client across execution contexts provides some performance enhancements due to reusing the underlying
// connection to the Redis cluster.
package redis

import (
	"fmt"
	"net"

	"com.aws-samples/apigateway.websockets.golang/lib/logger"
	"github.com/mediocregopher/radix/v3"
	"go.uber.org/zap"
)

// Client is the single client instance shared across the same Lambda execution contexts.
var Client *radix.Pool

func init() {
	cname, servers, err := net.LookupSRV("redis", "tcp", "service.internal")
	if err != nil {
		logger.Instance.Panic("unable to resolve redis srv record", zap.Error(err))
	}

	if len(servers) == 0 {
		logger.Instance.Panic("unable to resolve redis srv record")
	}

	logger.Instance.Info("redis srv record",
		zap.String("cname", cname),
		zap.Uint16("port", servers[0].Port),
		zap.String("target", servers[0].Target),
		zap.Uint16("weight", servers[0].Weight),
		zap.Uint16("priority", servers[0].Priority))

	addr := net.JoinHostPort(servers[0].Target, fmt.Sprintf("%d", servers[0].Port))
	Client, err = radix.NewPool("tcp", addr, 1)
	if err != nil {
		logger.Instance.Panic("unable to create redis connection pool", zap.Error(err))
	}
}
