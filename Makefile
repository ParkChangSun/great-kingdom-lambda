.PHONY: build

DIRS := $(wildcard handlers/RestApi/* handlers/WebSocketApi/*)
FUNCTIONS := $(notdir $(DIRS))

all: build

test:
	echo $(FUNCTIONS)

$(foreach func, $(FUNCTIONS), $(eval build-$(func): clean-$(func) ; $(MAKE) -C $(wildcard handlers/RestApi/$(func) handlers/WebSocketApi/$(func)) build))

$(foreach func, $(FUNCTIONS), $(eval clean-$(func): ; $(MAKE) -C $(wildcard handlers/RestApi/$(func) handlers/WebSocketApi/$(func)) clean))

build-GameTableEventFunc: clean-GameTableEventFunc
	$(MAKE) -C handlers/GameTableEventFunc build

clean-GameTableEventFunc:
	$(MAKE) -C handlers/GameTableEventFunc clean

build: 
	@echo "Building lambda functions..."
	@sam build

deploy: build
	@echo "Deploying application..."
	sam deploy