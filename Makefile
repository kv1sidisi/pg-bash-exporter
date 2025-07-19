.PHONY: build-linux build-windows

build-linux:
	@echo "Bilding for linux"
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o pg-bash-exporter-linux-amd64 ./cmd/exporter

build-windows:
	@echo "Building for windows"
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -v -o pg-bash-exporter-windows-amd64.exe ./cmd/exporter

