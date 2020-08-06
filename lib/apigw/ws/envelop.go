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

// Package ws provides common resources for working with Amazon API Gateway WebSockets
package ws

import "encoding/json"

// InputEnvelop defines the expected structure for incoming messages sent over the WebSocket connection. The envelop
// provides additional metadata in addition to the message data.
type InputEnvelop struct {
	Echo bool            `json:"echo"`
	Type int             `json:"type"`
	Data json.RawMessage `json:"data"`
}

// Decode decodes and populates the InputEnvelop from the provided bytes.
func (e *InputEnvelop) Decode(data []byte) (*InputEnvelop, error) {
	err := json.Unmarshal(data, e)
	return e, err
}

// OutputEnvelop defines the structure for messages sent over the WebSocket connection from the backend service. The
// envelop provides additional metadata in addition to the message data.
type OutputEnvelop struct {
	Type     int             `json:"type"`
	Data     json.RawMessage `json:"data"`
	Received int64           `json:"received"`
}

// Encode encodes the OutputEnvelop as JSON. The output is suitable for sending over the wire.
func (e *OutputEnvelop) Encode() ([]byte, error) {
	return json.Marshal(e)
}
