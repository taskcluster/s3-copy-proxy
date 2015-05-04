node_modules:
	npm install

.PHONY: node-test
node-test: node_modules
	npm test

go-test:
	godep go test

.PHONY: test
test: go-test node-test
