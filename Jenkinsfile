#!groovy

def GIT_BRANCH = ''
def IMAGE_REF = ''
def IMAGE_TAG = ''
def NOTIFIER_IMAGE = 'argo-rest-notifier'
def VERSION = ''
def NAMESPACE = ''

def runUtilityCommand(buildCommand) {
    // Run an arbitrary command inside the docker builder image
    sh "docker run -v ${pwd()}/dist:/go/src/github.com/cyrusbiotechnology/argo/dist --rm  builder-base:latest ${buildCommand}"
}

pipeline {
    agent any
    stages {
        stage('Checkout') {
            steps {
                checkout scm
                sh 'git submodule update --init --recursive'
                sh 'git rev-parse HEAD > git-sha.txt'
                script {
                    GIT_COMMIT = readFile 'git-sha.txt'
                    GIT_SHA = git.getCommit()
                    IMAGE_REF=docker2.imageRef()
                    IMAGE_TAG=IMAGE_REF.split(':').last()
                    GIT_BRANCH = env.BRANCH_NAME.replace('/', '').replace('_', '').replace('-', '')

                    def baseVersionTag = readFile "VERSION"
                    baseVersionTag = baseVersionTag.trim();
                    VERSION = "${baseVersionTag}-cyrus-${GIT_BRANCH}"

                    println "Version tag for this build is ${VERSION}"
                }
            }
        }

        stage('build utility container') {
            steps {
                sh "docker build -t builder-base --target builder-base ."
            }
        }


        stage('run tests') {
            steps {
                runUtilityCommand("go test ./...")
            }
        }


        stage('build controller') {
            steps {
                sh "docker build -t workflow-controller:${VERSION} --target workflow-controller ."
            }
        }

        stage('build executor') {
            steps {
                sh "docker build -t argoexec:${VERSION} --target argoexec ."
            }
        }




        stage('build Linux and MacOS CLIs') {
            steps {
                runUtilityCommand("make cli CGO_ENABLED=0  LDFLAGS='-extldflags \"-static\"' ARGO_CLI_NAME=argo-linux-amd64")
                runUtilityCommand("make cli CGO_ENABLED=0  LDFLAGS='-extldflags \"-static\"' ARGO_CLI_NAME=argo-darwin-amd64")
            }
        }

        stage('push containers to GCR') {

            steps {
                script { docker2.push("workflow-controller:${VERSION}", ["workflow-controller:${VERSION}"]) }
                script { docker2.push("argoexec:${VERSION}", ["argoexec:${VERSION}"]) }

            }

        }

        stage('push CLI to artifactory') {
            steps {
                withCredentials([usernamePassword(credentialsId: 'Artifactory', usernameVariable: 'ARTI_NAME', passwordVariable: 'ARTI_PASS')]) {
                    runUtilityCommand("curl -u ${ARTI_NAME}:${ARTI_PASS} -T /go/src/github.com/cyrusbiotechnology/argo/dist/argo-darwin-amd64 https://cyrusbio.jfrog.io/cyrusbio/argo-cli/argo-mac-${VERSION}")
                    runUtilityCommand("curl -u ${ARTI_NAME}:${ARTI_PASS} -T /go/src/github.com/cyrusbiotechnology/argo/dist/argo-linux-amd64 https://cyrusbio.jfrog.io/cyrusbio/argo-cli/argo-linux-${VERSION}")
                }
            }
        }

    }
 }
