.PHONY: build

DIRS := $(wildcard handlers/RestApi/* handlers/WebSocketApi/*)
FUNCTIONS := $(notdir $(DIRS))

all: build

$(foreach func, $(FUNCTIONS), $(eval build-$(func): ; $(MAKE) -C $(wildcard handlers/RestApi/$(func) handlers/WebSocketApi/$(func)) build))

build-GameTableEventFunc:
	$(MAKE) -C handlers/GameTableEventFunc build

build: 
	@echo "Building lambda functions..."
	@sam build

deploy: build
	@echo "Deploying application..."
	sam deploy