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

// Package apigw provides common resources for working with Amazon API Gateway
package apigw

import (
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
)

// NewAPIGatewayManagementClient creates a new API Gateway Management Client instance from the provided parameters. The
// new client will have a custom endpoint that resolves to the application's deployed API.
func NewAPIGatewayManagementClient(cfg *aws.Config, domain, stage string) *apigatewaymanagementapi.Client {
	cp := cfg.Copy()
	cp.EndpointResolver = aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		if service != "execute-api" {
			return cfg.EndpointResolver.ResolveEndpoint(service, region)
		}

		var endpoint url.URL
		endpoint.Path = stage
		endpoint.Host = domain
		endpoint.Scheme = "https"
		return aws.Endpoint{
			SigningRegion: region,
			URL:           endpoint.String(),
		}, nil
	})

	return apigatewaymanagementapi.New(cp)
}
