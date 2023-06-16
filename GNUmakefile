all: build_deps all_artifacts

GOBIN=$(shell echo $$(go env GOPATH)/bin)

##############################################################################
# Top level targets

all_artifacts: binaries schema.json


binaries: srs

dist/bffd: cmd/bffd/bffd.go
	go build -o $@ ./$<


.PHONY: format
format: build_deps
	go fmt ./...
	go vet ./...
	$(GOBIN)/staticcheck ./...
	$(GOBIN)/gosimports -w -local $(shell head -1 go.mod | awk '{print $$2}') ./cmd ./internal

clean:
	rm -rf .d

##############################################################################
# Dependencies

build_deps: build .d .d/graphql .d/tidy .d/tools

.d:
	mkdir .d
.d/tidy: go.mod tools.go
	go mod tidy
	touch .d/tidy
.d/tools: tools.go
	cat internal/tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -tI % go install %
	touch .d/tools



upgrade-all:
	go get -u ./...
	$(MAKE) upgrade-tools

upgrade-tools:
	cat internal/tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -tI % go get -u %

build:
	mkdir build

deploy:
	gcloud config configurations activate pp
	gcloud builds submit --config scraper.yaml
	gcloud run deploy scraper \
			--image gcr.io/peak-profits/scraper \
			--platform=managed \
			--region us \
			--project peak-profits


# disables all implicit rules
.SUFFIXES:
