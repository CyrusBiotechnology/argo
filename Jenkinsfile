#!groovy

def GIT_BRANCH = ''
def IMAGE_REF = ''
def IMAGE_TAG = ''
def NOTIFIER_IMAGE = 'argo-rest-notifier'
def VERSION = ''
def NAMESPACE = ''

def runUtilityCommand(buildCommand) {
    // Run an arbitrary command inside the docker builder image
    sh "docker run --rm " +
       "-v ${pwd()}/dist/pkg:/root/go/pkg " +
       "-v ${pwd()}:/root/go/src/github.com/CyrusBiotechnology/argo " +
       "-w /root/go/src/github.com/CyrusBiotechnology/argo argo-builder ${buildCommand}"
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
                sh "docker build -t argo-builder -f Dockerfile-builder ."
            }
        }

        stage('run tests') {
            steps {
                runUtilityCommand("go test ./...")
            }
        }

        stage('build controller') {
            steps {
                runUtilityCommand("make controller")
                sh "docker build -t workflow-controller:${VERSION} -f Dockerfile-workflow-controller ."
            }
        }

        stage('build executor') {
            steps {
                runUtilityCommand("make executor")
                sh "docker build -t argoexec:${VERSION} -f Dockerfile-argoexec ."
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
                    runUtilityCommand("curl -u ${ARTI_NAME}:${ARTI_PASS} -T /root/go/src/github.com/CyrusBiotechnology/argo/dist/argo-darwin-amd64 https://cyrusbio.jfrog.io/cyrusbio/argo-cli/argo-mac-${VERSION}")
                    runUtilityCommand("curl -u ${ARTI_NAME}:${ARTI_PASS} -T /root/go/src/github.com/CyrusBiotechnology/argo/dist/argo-linux-amd64 https://cyrusbio.jfrog.io/cyrusbio/argo-cli/argo-linux-${VERSION}")
                }
            }
        }

        stage('Deploy') {
            steps {
                script {
                    if (GIT_BRANCH in ['rc', 'release', 'master_engineering']) {
                        NAMESPACE = k8s.getNamespaceFromBranch(GIT_BRANCH) ?: 'development'
                        k8s.updateImageTag(NAMESPACE, docker2.imageTag(), 'gcr.io/cyrus-containers/argo-rest', GIT_BRANCH)
                    }
                }
            }
        }

    }
 }