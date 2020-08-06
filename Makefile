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

root	:=		$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

.PHONY: check clean test build deploy lint install-golangci

install-golangci:
	wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.27.0

lint:
	"${root}/bin/golangci-lint" run 
	cfn-lint --template "${root}/template.yml"
	cfn_nag "${root}/template.yml" --blacklist-path "${root}/.cfnnag.yml"

test:
	go test -v ./...

clean:
	$(MAKE) -C "${root}/publish" clean
	$(MAKE) -C "${root}/connect" clean
	$(MAKE) -C "${root}/disconnect" clean

build: clean
	@echo "building handlers for aws lambda"
	$(MAKE) -C "${root}/publish" build
	$(MAKE) -C "${root}/connect" build
	$(MAKE) -C "${root}/disconnect" build

deploy: check
	@echo "deploying infrastructure and code"
	sam package --output-template-file "${root}"/packaged.yml --s3-bucket "${bucket}"
	sam deploy --stack-name "${stack}" --capabilities CAPABILITY_IAM --template-file "${root}/packaged.yml"

check:
ifndef bucket
	$(error bucket was not provided)
endif
ifndef stack
	$(error stack was not provided)
endif
