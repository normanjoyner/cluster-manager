def loadUtils() {
    return fileLoader.fromGit('Jenkinsfile', 'https://github.com/containership/containership.cloud.jenkins-pipeline.git', 'master', 'containershipbot_github_user_pass', '')
}

def utils

node {
    stage('Preparation') {
        utils = loadUtils()
    }
}

runPipeline()

def runPipeline() {
    try {
        runStages();
    } catch(err) {
        echo "Error: ${err}"
        currentBuild.result = "FAILED"
    }
}

def runStages() {
	utils = loadUtils()
    def buildInfo = [
        config: null,
        environment: 'default',
        image: [
            name: "${env.BUILD_TAG}",
            tag: "latest"
        ],
        scm: [
            branch: null,
            sha: null,
            tag: null,
            isPR: false
        ],
        deploy: [
            userInput: false
        ],
        isFeatureBranch: false
    ]

    node {
        try {
            stage('Build') {
                checkout scm

                def githubInfo = utils.calculateGithubInfo()
                buildInfo.scm.branch = githubInfo.branch
                buildInfo.scm.sha = githubInfo.sha
                buildInfo.scm.tag = githubInfo.tag
                buildInfo.scm.isPR = githubInfo.isPR
                buildInfo.isFeatureBranch = buildInfo.scm.branch.contains('feature/')
                buildInfo.config = utils.getContainershipJenkinsConfig()

                def env = utils.getEnvironment(buildInfo.scm.branch, buildInfo.config)
                buildInfo.environment = env

                println "Build environment: ${env}"

                buildInfo.config = buildInfo.config["${env}"]

                if(!buildInfo.config) {
                    throw new Exception("You must have a valid .containership-jenkins.json file in your repository")
                }

                if (buildInfo.config.containership != null) {
                    buildInfo.config.containership.'add_loadbalancer' = buildInfo.config.containership.'add_loadbalancer' || false
                }

				def branchNameVariables = utils.buildBranchVariableMap(buildInfo.config.branch, buildInfo.scm.branch)

                // replace all dynamic vars
                for (def entry in utils.mapToList(branchNameVariables)) {
                    if (buildInfo.config.docker.image['tag-postfix']) {
                        buildInfo.config.docker.image['tag-postfix'] = buildInfo.config.docker.image['tag-postfix'].replace(entry.key, entry.value)
                    }

                    if (buildInfo.config.containership.application) {
                        buildInfo.config.containership.application = buildInfo.config.containership.application.replace(entry.key, entry.value)
                    }
                }

                buildInfo.deploy.userInput = buildInfo.config.containership?.input?.type == 'user'

                def imageInfo = utils.calculateImageInfo(buildInfo)
                buildInfo.image.name = "${imageInfo.owner}/${imageInfo.name}"
                buildInfo.image.tag = imageInfo.tag

                if(imageInfo.'tag-postfix' != null) {
                    buildInfo.image.tag += "-${imageInfo.'tag-postfix'}"
                }

                buildInfo.image.'calculated-name' = "${buildInfo.image.name}:${buildInfo.image.tag}"

                if(!fileExists(buildInfo.config.docker?.dockerfile ?: 'Dockerfile.coordinator')) {
                    createGoDockerfile(buildInfo.config.docker?.dockerfile ?: 'Dockerfile.coordinator')
                }

                utils.buildDockerfile(buildInfo.config.docker?.dockerfile ?: 'Dockerfile.coordinator', "${buildInfo.image.'calculated-name'}", buildInfo.config.docker?.'build-args')
            }

            parallel(
                lint: {
                    stage('Linting') {
                        utils.runCmdOnDockerImageWithEntrypoint(buildInfo.image.'calculated-name', 'sh', '-c "cd /gocode/src/github.com/containership/cloud-agent/ && go get -u github.com/golang/lint/golint && PATH=$PATH:/gocode/bin && make lint"')
                    }
                },
                test: {
                    stage('Testing') {
                        utils.runCmdOnDockerImageWithEntrypoint(buildInfo.image.'calculated-name', 'sh', '-c "cd /gocode/src/github.com/containership/cloud-agent/ && make test"')
                    }
                },
                vet: {
                    stage('Vet') {
                        utils.runCmdOnDockerImageWithEntrypoint(buildInfo.image.'calculated-name', 'sh', '-c "cd /gocode/src/github.com/containership/cloud-agent/ && make vet"')
                    }
                },
                format: {
                    stage('Formating') {
                        utils.runCmdOnDockerImageWithEntrypoint(buildInfo.image.'calculated-name', 'sh', '-c "cd /gocode/src/github.com/containership/cloud-agent/ && ! gofmt -d -s internal pkg cmd 2>&1 | read"')
                    }
                }
            )

            stage('Cleanup') {
                utils.cleanupImage(buildInfo.image.name, buildInfo.image.tag)
            }
        } catch(err) {
            utils.cleanupImage(buildInfo.image.name, buildInfo.image.tag)
            throw err
        }
    }
}

def createGoDockerfile(path) {
    def docker = """
####
# Dockerfile for Containership Agent
####
FROM iron/go:dev
MAINTAINER ContainerShip Developers <developers@containership.io>
# add tools for debug and development purposes
RUN apk update && apk add vim && apk add iptables && apk add glide

ENV SRC_DIR=/gocode/src/github.com/containership/cloud-agent/

WORKDIR /app

# Add the source code:
ADD . $SRC_DIR

# Build it:
RUN cd $SRC_DIR && \
    glide install && \
    go build -o coordinator cmd/cloud_coordinator/main.go && \
    cp coordinator /app

ENTRYPOINT /app/coordinator

ARG NPM_TOKEN
RUN mkdir /app
ADD . /app
WORKDIR /app
RUN npm install yarn -g
RUN yarn --version
RUN echo "//registry.npmjs.org/:_authToken=\$NPM_TOKEN" > .npmrc
RUN yarn install --ignore-engines
CMD node application
"""

    sh "echo \"${docker}\" > ${path}"
}
