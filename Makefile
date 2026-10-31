.PHONY: build
build:
	go build

.PHONY: test
test:
	go vet ./...

.PHONY: clean
clean:
	$(RM) nanashi
