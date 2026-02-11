# otel-databricks-collector

otel-databricks-collector is a utility project for exporting Databricks data and integrating it with OpenTelemetry (OTel) workflows.

It provides tooling to run notebooks, extract table data, and forward telemetry or structured data to downstream systems.

---

## Overview

This project is designed to:

- Execute Databricks notebooks programmatically
- Export table data from Databricks
- Integrate with OpenTelemetry pipelines
- Support CI/CD workflows via Jenkins
- Provide a structured Make-based development setup

---

## Project Structure

- `tableexporter/` – Logic for exporting Databricks tables  
- `otel_run_notebook.py` – Script to trigger and manage notebook execution  
- `Makefile` – Development and automation commands  
- `Jenkinsfile` – CI/CD pipeline configuration  
- `.gitignore` – Git configuration  

---

## Requirements

- Python 3.9+
- Databricks workspace access
- Databricks API token
- OpenTelemetry SDK (if exporting traces/metrics)
- Make (optional)

Install dependencies (if requirements file exists):

```bash
pip install -r requirements.txt
```

---

## Configuration

Set required environment variables:

```bash
export DATABRICKS_HOST=https://<your-databricks-instance>
export DATABRICKS_TOKEN=<your-databricks-token>
```

If OpenTelemetry integration is enabled:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=<collector-endpoint>
```

---

## Running Notebook Export

Trigger a Databricks notebook run:

```bash
python otel_run_notebook.py --notebook-path "/Workspace/Path/To/Notebook"
```

Additional arguments may be supported depending on implementation.

---

## CI/CD

The repository includes a `Jenkinsfile` for pipeline automation.  
It can be used to:

- Run notebook exports
- Validate data extraction
- Integrate with telemetry pipelines

---

## Development

Common commands (via Makefile):

```bash
make run
make build
make test
```

(Adjust according to Makefile definitions.)

---

## Use Cases

- Exporting Databricks tables for monitoring
- Automating n
