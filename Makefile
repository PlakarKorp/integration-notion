all: importer exporter

importer:
	go build -o notion-importer -v ./importer

exporter:
	go build -o notion-exporter -v ./exporter

.PHONY: all importer exporter
