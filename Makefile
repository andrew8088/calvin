.PHONY: build dev

CALVIN_OAUTH_CLIENT_ID ?= $(shell echo $$CALVIN_OAUTH_CLIENT_ID)
CALVIN_OAUTH_CLIENT_SECRET ?= $(shell echo $$CALVIN_OAUTH_CLIENT_SECRET)

build:
	go build -ldflags "\
		-X 'github.com/andrew8088/calvin/internal/auth.embeddedClientID=$(CALVIN_OAUTH_CLIENT_ID)' \
		-X 'github.com/andrew8088/calvin/internal/auth.embeddedClientSecret=$(CALVIN_OAUTH_CLIENT_SECRET)'" \
		-o calvin .

dev:
	go build -o calvin .
