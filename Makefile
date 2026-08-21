GC=go build

bin/flippyram-server: main.go $(shell find src -type f -name "*.go")
	$(GC) -o $@ $<

.PHONY: cleanall
cleanall:
	-rm -f bin/*
