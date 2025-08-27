# Databricks notebook source
# MAGIC %sh curl -o otelcol-custom https://artifactory.nike.com:9002/artifactory/python-virtual/otelcol-custom-1.0+pr2.4
# MAGIC

# COMMAND ----------

# Get databricks token from notebook
databricks_token = dbutils.notebook.entry_point.getDbutils().notebook().getContext().apiToken().getOrElse(None)
databricks_host = "nike-sole-react.cloud.databricks.com"
# Get secret from Databricks secret scope
secret_scope = "test_chase_allen"
token_secret_key = "noop-token-global-test"
cert_secret_key = "noop-ca-cert-global-test"
noop_token = dbutils.secrets.get(scope=secret_scope, key=token_secret_key)
noop_cert = dbutils.secrets.get(scope=secret_scope, key=cert_secret_key)
import os
os.environ['DATABRICKS_TOKEN'] = databricks_token
os.environ['NOOP_METRICS_TOKEN'] = noop_token
os.environ['NOOP_METRICS_CA_CERT'] = noop_cert


# COMMAND ----------

# Set up service discovery for endpoints
import requests
import yaml

# Set up the API endpoint and headers
url = f"https://{databricks_host}/api/2.0/serving-endpoints"
headers = {
    "Authorization": f"Bearer {databricks_token}"
}

# Make the request to list all serving endpoints
response = requests.get(url, headers=headers)
endpoints = response.json()["endpoints"]
static_configs = []

for e in endpoints:
    if e.get("endpoint_type", "") != "FOUNDATION_MODEL_API":
        static_configs.append({"targets": [f"{databricks_host}"],
                                "labels":{"__metrics_path__": f"/api/2.0/serving-endpoints/{e['name']}/metrics"}})
yaml_config = yaml.dump(static_configs)
# write to a file with yaml config
with open('otel-targets.yaml', 'w') as f:
    f.write(yaml_config)

# COMMAND ----------

# MAGIC %sh sudo apt-get update

# COMMAND ----------

# MAGIC %sh sudo apt-get -y install wget

# COMMAND ----------

# MAGIC %sh [ -f otelcol_0.122.1_linux_amd64.deb ] || wget https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/v0.122.1/otelcol_0.122.1_linux_amd64.deb

# COMMAND ----------

# MAGIC %sh sudo dpkg -i otelcol_0.122.1_linux_amd64.deb

# COMMAND ----------

# MAGIC %%sh
# MAGIC echo $NOOP_METRICS_CA_CERT > otel-global-test-ca.pem
# MAGIC sed -i 's/\\n/\n/g' otel-global-test-ca.pem

# COMMAND ----------

# MAGIC %%writefile otel-config.yaml
# MAGIC receivers:
# MAGIC   prometheus:
# MAGIC     config:
# MAGIC       scrape_configs:
# MAGIC         - job_name: 'databricks-endpoint'
# MAGIC           scrape_interval: 30s
# MAGIC           scheme: https
# MAGIC           file_sd_configs:
# MAGIC           - files:
# MAGIC             - 'otel-targets.yaml'
# MAGIC           relabel_configs:
# MAGIC             - source_labels: [__metrics_path__]
# MAGIC               target_label: metrics_path
# MAGIC           authorization:
# MAGIC             credentials: "${DATABRICKS_TOKEN}"
# MAGIC
# MAGIC processors:
# MAGIC   transform/metrics_path_to_resource:
# MAGIC     error_mode: ignore
# MAGIC     metric_statements:
# MAGIC       - context: datapoint
# MAGIC         statements:
# MAGIC           - set(resource.attributes["metrics_path"], attributes["metrics_path"])
# MAGIC           - delete_key(attributes, "metrics_path")
# MAGIC
# MAGIC exporters:
# MAGIC   tableexporter:
# MAGIC     db_host: "nike-sole-react.cloud.databricks.com"
# MAGIC     db_http_path: "/sql/1.0/warehouses/b0049f9f7b514584"
# MAGIC     db_token: "${DATABRICKS_TOKEN}"
# MAGIC     target_table: "development.james_rag.otel_metrics_chase_allen"
# MAGIC
# MAGIC   debug:
# MAGIC     verbosity: detailed
# MAGIC
# MAGIC   otlp/metrics:
# MAGIC     endpoint: "https://observability.global-observability-test.nikecloud.com:4317"
# MAGIC     compression: gzip
# MAGIC     tls:
# MAGIC       insecure: false
# MAGIC       ca_file: "otel-global-test-ca.pem"
# MAGIC     headers:
# MAGIC       X-Scope-OrgID: "internal"
# MAGIC       Authorization: "${NOOP_METRICS_TOKEN}"
# MAGIC       X-Environment: "test"
# MAGIC
# MAGIC service:
# MAGIC   pipelines:
# MAGIC     metrics:
# MAGIC       receivers: [prometheus]
# MAGIC       processors: [transform/metrics_path_to_resource]
# MAGIC       exporters: [tableexporter]
# MAGIC   telemetry:
# MAGIC     logs:
# MAGIC       level: debug
# MAGIC     metrics:
# MAGIC       readers:
# MAGIC         - pull:
# MAGIC             exporter:
# MAGIC               prometheus:
# MAGIC                 host: '0.0.0.0'
# MAGIC                 port: 8889

# COMMAND ----------

# MAGIC %sh ./otelcol-custom --config=file:otel-config.yaml
# MAGIC # import subprocess
# MAGIC
# MAGIC # otel_process = subprocess.Popen(["otelcol-custom", "--config=file:otel-config.yaml"], stdout=subprocess.PIPE, stderr=subprocess.PIPE)
# MAGIC # otel_process.kill()
