.PHONY: build

DIRS := $(wildcard handlers/RestApi/* handlers/WebSocketApi/*)
FUNCTIONS := $(notdir $(DIRS))

all: build

$(foreach func, $(FUNCTIONS), $(eval build-$(func): ; $(MAKE) -C $(wildcard handlers/RestApi/$(func) handlers/WebSocketApi/$(func)) build))

build-GameTableHandlerFunc:
	$(MAKE) -C handlers/GameTableHandlerFunc build

build: 
	@sam build -p

clean:
	@for f in $(DIRS) ; do $(MAKE) -C $$f clean ; done
	$(MAKE) -C handlers/GameTableHandlerFun clean