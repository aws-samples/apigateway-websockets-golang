# MIT No Attribution

# Copyright 2020 Amazon.com, Inc. or its affiliates.

# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:

# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.

# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.

.PHONY: check clean test build deploy

test:
	go test -v ./...

clean:
	$(MAKE) -C publish clean
	$(MAKE) -C connect clean
	$(MAKE) -C disconnect clean

build: clean
	@echo "building handlers for aws lambda"
	sam build

build-ConnectFunction:
	@echo "building handler for aws lambda"
	$(MAKE) -C connect build

build-DisconnectFunction:
	@echo "building handler for aws lambda"
	$(MAKE) -C disconnect build

build-PublishFunction:
	@echo "building handler for aws lambda"
	$(MAKE) -C publish build

deploy: check
	@echo "deploying infrastructure and code"
	sam package --output-template-file packaged.yml --s3-bucket "${bucket}"
	sam deploy --no-fail-on-empty-changeset \
		--stack-name "${stack}" \
		--template-file packaged.yml \
		--capabilities CAPABILITY_IAM

check:
ifndef bucket
	$(error bucket was not provided)
endif
ifndef stack
	$(error stack was not provided)
endif
