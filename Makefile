.PHONY: build install uninstall clean

BINARY_NAME=nsh
INSTALL_PATH=/usr/local/bin

build:
	go build -o $(BINARY_NAME) ./cmd/nsh

install: build
	sudo cp $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✓ Installed $(BINARY_NAME) to $(INSTALL_PATH)"

uninstall:
	sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✓ Uninstalled $(BINARY_NAME)"

clean:
	rm -f $(BINARY_NAME)
