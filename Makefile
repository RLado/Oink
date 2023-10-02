BINARY_NAME=oink
 
build:
	@go build -o ${BINARY_NAME} src/main.go

clean:
	@go clean
	@rm ${BINARY_NAME}

install: build
# check the user is root
	@if [ `id -u` -ne 0 ]; then echo "Please run as root"; exit 1; fi

# install binary
	@cp ${BINARY_NAME} /usr/bin/${BINARY_NAME}
	@chmod 755 /usr/bin/${BINARY_NAME}

# install configuration file
	@mkdir -p /etc/oink_ddns/
	@cp config/config.json /etc/oink_ddns/config.json
	@chmod 600 /etc/oink_ddns/config.json

# install systemd service
	@cp config/oink_ddns.service /lib/systemd/system/oink_ddns.service
	@chmod 644 /lib/systemd/system/oink_ddns.service

# advice the user
	@echo "\033[38;2;255;133;162mOink installed successfully\033[0m"
	@echo "Please remember to edit /etc/oink_ddns/config.json before enabling the DDNS client using 'systemctl enable oink_ddns.service' 'systemctl start oink_ddns.service'"

uninstall:
# check the user is root
	@if [ `id -u` -ne 0 ]; then echo "Please run as root"; exit 1; fi

# completely remove the binary and configuration file
	@rm /usr/bin/${BINARY_NAME}
	@rm /etc/oink_ddns/config.json

# remove systemd service
	@systemctl stop oink_ddns.service
	@systemctl disable oink_ddns.service
	@rm /lib/systemd/system/oink_ddns.service

# notify the user
	@echo "\033[38;2;255;133;162mOink uninstalled successfully\033[0m"
