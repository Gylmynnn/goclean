# Makefile for goclean

BINARY_NAME := goclean
INSTALL_DIR := /usr/local/bin

.PHONY: build install uninstall clean

build:
	go build -o $(BINARY_NAME) .

install: build
	sudo cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	sudo chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "$(BINARY_NAME) installed to $(INSTALL_DIR)"

uninstall:
	sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "$(BINARY_NAME) uninstalled from $(INSTALL_DIR)"

clean:
	rm -f $(BINARY_NAME)
