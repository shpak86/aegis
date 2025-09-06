# Simple Makefile for nginx-aegis module
# Compatible with nginx 1.24+

# Variables
NGINX_VERSION ?= 1.24.0
MODULE_DIR = $(shell pwd)
BUILD_DIR = $(MODULE_DIR)/build

# Default target
all: build

# Clean and build
build: clean download configure compile show-path

# Download nginx source
download:
	@echo "📥 Downloading nginx $(NGINX_VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@cd $(BUILD_DIR) && \
		wget -q -O nginx-$(NGINX_VERSION).tar.gz \
		"http://nginx.org/download/nginx-$(NGINX_VERSION).tar.gz" && \
		tar -xzf nginx-$(NGINX_VERSION).tar.gz

# Configure nginx
configure:
	@echo "⚙️ Configuring nginx with aegis module..."
	@cd $(BUILD_DIR)/nginx-$(NGINX_VERSION) && \
		./configure \
			--add-dynamic-module=$(MODULE_DIR) \
			--with-compat \
			--with-file-aio \
			--with-http_ssl_module \
			--with-http_v2_module \
			--with-http_gzip_static_module \
			--with-http_secure_link_module \
			--with-http_stub_status_module

# Compile module
compile:
	@echo "🔨 Building module..."
	@cd $(BUILD_DIR)/nginx-$(NGINX_VERSION) && make modules

# Show module path
show-path:
	@echo ""
	@echo "✅ Module built successfully!"
	@echo ""
	@echo "📦 Module location:"
	@echo "   $(BUILD_DIR)/nginx-$(NGINX_VERSION)/objs/ngx_http_aegis_module.so"
	@echo ""
	@if [ -f "$(BUILD_DIR)/nginx-$(NGINX_VERSION)/objs/ngx_http_aegis_module.so" ]; then \
		ls -la "$(BUILD_DIR)/nginx-$(NGINX_VERSION)/objs/ngx_http_aegis_module.so"; \
	fi

# Clean build directory
clean:
	@echo "🧹 Cleaning build directory..."
	@rm -rf $(BUILD_DIR)

# Install module to common nginx paths
install: build
	@echo "📋 Installing module to system paths..."
	@MODULE_FILE="$(BUILD_DIR)/nginx-$(NGINX_VERSION)/objs/ngx_http_aegis_module.so"; \
	if [ -f "$$MODULE_FILE" ]; then \
		for path in /usr/lib/nginx/modules /usr/share/nginx/modules /etc/nginx/modules; do \
			if [ -d "$$path" ] || sudo mkdir -p "$$path" 2>/dev/null; then \
				echo "  Installing to $$path/"; \
				sudo cp "$$MODULE_FILE" "$$path/"; \
			fi; \
		done; \
		echo ""; \
		echo "✅ Installation complete!"; \
		echo ""; \
		echo "Add to nginx.conf:"; \
		echo "  load_module modules/ngx_http_aegis_module.so;"; \
	else \
		echo "❌ Module file not found!"; \
		exit 1; \
	fi

# Help
help:
	@echo "Available targets:"
	@echo "  build        - Download, configure and build module (default)"
	@echo "  install      - Build and install module to system paths"
	@echo "  clean        - Clean build directory"
	@echo "  help         - Show this help"
	@echo ""
	@echo "Environment variables:"
	@echo "  NGINX_VERSION - nginx version to build against (default: 1.24.0)"
	@echo ""
	@echo "Examples:"
	@echo "  make                          # Build with nginx 1.24.0"
	@echo "  NGINX_VERSION=1.26.0 make     # Build with nginx 1.26.0"
	@echo "  make install                  # Build and install"

.PHONY: all build download configure compile show-path clean install help
