# Change these variables as necessary.
MAIN_PACKAGE_PATH := ./cmd/main.go
BINARY_NAME := blogs_server

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## tidy: format code and tidy modfile
.PHONY: tidy
tidy:
	go fmt ./...
	go mod tidy -v

## audit: run quality control checks
.PHONY: audit
audit:
	go mod verify
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	go test -race -buildvcs -vet=off ./...


# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## test: run all tests
.PHONY: test
test:
	go test -v -race -buildvcs ./...

## test/cover: run all tests and display coverage
.PHONY: test/cover
test/cover:
	go test -v -race -buildvcs -coverprofile=/tmp/coverage.out ./...
	go tool cover -html=/tmp/coverage.out

## generate_cert: generate_cert
.PHONY: generate_cert
generate_cert:
	go run generate_cert.go --host localhost

## build: build the application
.PHONY: build
build:
	# Include additional build steps, like TypeScript, SCSS or Tailwind compilation here...
	~/go/bin/templ generate && go build -o=./tmp/bin/${BINARY_NAME} ${MAIN_PACKAGE_PATH}

.PHONY: db
db:
	docker run --name wondertales_blogs_db --rm -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=123 -e PGDATA=/var/lib/postgresql/data/pgdata -v /tmp:/var/lib/postgresql/data -p 5432:5432 -it postgres:14.1-alpine

## run: run the application with reloading on file changes
.PHONY: run
run:
	go run github.com/cosmtrek/air@v1.43.0 \
		--build.cmd "make build" --build.bin "LOCAL=1 ./tmp/bin/${BINARY_NAME}" --build.delay "100" \
		--build.exclude_dir "" \
		--build.include_ext "go, tpl, templ, html, css, js, ts, sql, jpeg, jpg, gif, png, bmp, svg, webp, ico" \
		--build.exclude_regex "_templ.go"
		--misc.clean_on_exit "true"


# ==================================================================================== #
# OPERATIONS
# ==================================================================================== #

## push: push changes to the remote Git repository
.PHONY: push
push: tidy audit no-dirty
	git push

## production/deploy: deploy the application to production
.PHONY: production/deploy
production/deploy: confirm tidy audit no-dirty
	GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=/tmp/bin/linux_amd64/${BINARY_NAME} ${MAIN_PACKAGE_PATH}
	upx -5 /tmp/bin/linux_amd64/${BINARY_NAME}
	# Include additional deployment steps here...