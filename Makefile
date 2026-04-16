.PHONY: build dev release

CALVIN_OAUTH_CLIENT_ID ?= $(shell echo $$CALVIN_OAUTH_CLIENT_ID)
CALVIN_OAUTH_CLIENT_SECRET ?= $(shell echo $$CALVIN_OAUTH_CLIENT_SECRET)

ifeq ($(firstword $(MAKECMDGOALS)),release)
RELEASE_VERSION_ARG := $(word 2,$(MAKECMDGOALS))
ifneq ($(RELEASE_VERSION_ARG),)
$(eval $(RELEASE_VERSION_ARG):;@:)
endif
endif

build:
	go build -ldflags "\
		-X 'github.com/andrew8088/calvin/internal/auth.embeddedClientID=$(CALVIN_OAUTH_CLIENT_ID)' \
		-X 'github.com/andrew8088/calvin/internal/auth.embeddedClientSecret=$(CALVIN_OAUTH_CLIENT_SECRET)'" \
		-o calvin .

dev:
	go build -o calvin .

release:
	./scripts/release.sh "$(or $(VERSION),$(RELEASE_VERSION_ARG))"
