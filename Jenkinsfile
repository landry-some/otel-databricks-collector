@Library('cop-pipeline-step') _

def pyimage = docker.image("artifactory.nike.com:9002/aiml/aiml-poetry-docker:master-1")

MAJOR_VERSION = "1.0"

if (env.BRANCH_NAME != 'master') {
    DEV_VERSION = env.BRANCH_NAME.replaceAll("[^a-zA-Z0-9]+","").toLowerCase()
    MAJOR_VERSION = MAJOR_VERSION + "+${DEV_VERSION}"
}
LIB_VERSION="${MAJOR_VERSION}.${env.BUILD_NUMBER}"
echo 'LIB_VERSION=' + LIB_VERSION

ARTIFACTORY_URL = "artifactory.nike.com:9002"

pipeline {
    agent any
    options {
      buildDiscarder logRotator(numToKeepStr: '10')
      disableConcurrentBuilds()
    }

    stages {
        stage('scm') {
            steps {
                script {
                    cleanWs()
                    checkout([
                        $class: 'GitSCM',
                        branches: scm.branches,
                        userRemoteConfigs: scm.userRemoteConfigs
                    ])
                }
            }
        }

        stage('Install Libraries and Build') {
            steps {
                script {
                    pyimage.inside {
                        sh """
                        WORKSPACE_ROOT=\$PWD

                        export POETRY_CACHE_DIR="\$PWD/.cache/pypoetry"
                        export POETRY_HOME="\$PWD/.local/share/pypoetry"

                        mkdir -p go-install .gocache .gomodcache .go

                        wget https://go.dev/dl/go1.23.8.linux-amd64.tar.gz
                        tar -C ./go-install -xzf go1.23.8.linux-amd64.tar.gz
                        rm -rf ./go-install/go/test

                        export PATH="\$PWD/go-install/go/bin:\$PATH"
                        export GOCACHE="\$PWD/.gocache"
                        export GOMODCACHE="\$PWD/.gomodcache"
                        export GOPATH="\$PWD/.go"
                        export GO111MODULE=on
                        export GOTOOLCHAIN=local

                        go version

                        ls
                        cd \$WORKSPACE_ROOT
                        ls

                        cd tableexporter
                        go mod tidy

                        go install go.opentelemetry.io/collector/cmd/builder@v0.124.0
                        export PATH="\$GOPATH/bin:\$PATH"
                        builder version

                        cd ..

                        make build

                        """
                    }
                }
            }
        }

        stage('upload to artifactory') {
            steps {
                withCerberus([sdbPath: 'app/aiml-platform/artifactory',
                    sdbKeys: ['username': 'ARTIFACTORY_USERNAME', 'password': 'ARTIFACTORY_PASSWORD']
                ]){ secrets ->
                    script {
                        pyimage.inside {
                            sh """
                            # Build path and filename
                            DIST_FILE="dist/otel-custom"
                            ARTIFACTORY_REPO="python-virtual"
                            ARTIFACTORY_UPLOAD_URL="https://${ARTIFACTORY_URL}/artifactory/\$ARTIFACTORY_REPO/otelcol-custom-${LIB_VERSION}"
                            
                            echo "Uploading \$DIST_FILE to \$ARTIFACTORY_UPLOAD_URL"
                            ls
                            ls dist

                            chmod 644 dist/otelcol-custom

                            curl -u '${ARTIFACTORY_USERNAME}:${ARTIFACTORY_PASSWORD}' -X PUT --data-binary @dist/otelcol-custom "\$ARTIFACTORY_UPLOAD_URL"
                            """
                        }
                    }
                }
            }
        }
    }
}
