build:
	builder  --config=tableexporter/otelcol-builder.yaml

run:
	dist/otelcol-custom --config=local-test/otelcol-config.yaml

install:
	go mod tidy -e
