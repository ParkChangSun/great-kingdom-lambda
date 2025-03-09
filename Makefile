.PHONY: build clean

SUBDIRS=handler

clean:


build:
	@echo "building handlers for aws lambda"
	sam build
