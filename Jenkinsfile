@Library('containership-jenkins@v3')
import io.containership.*

def dockerUtils = new Docker(this)
def gitUtils = new Git(this)
def npmUtils = new Npm(this)
def pipelineUtils = new Pipeline(this)
def kubectlUtils = new Kubectl(this)

pipelineUtils.jenkinsWithNodeTemplate {
    def docker_org = 'containership'
    def docker_name_agent = 'cloud-agent'
    def docker_name_coordinator = 'cloud-coordinator'

    def dockerfile_test_agent = 'Dockerfile.test'
    def dockerfile_test_coordinator = 'Dockerfile.test'

    def dockerfile_agent = 'Dockerfile-jenkins.agent'
    def dockerfile_coordinator = 'Dockerfile-jenkins.coordinator'

    def docker_repo_agent = "${docker_org}/${docker_name_agent}"
    def docker_repo_coordinator = "${docker_org}/${docker_name_coordinator}"

    def deploy_branch = 'master'

    // will be set in checkout stage
    def git_commit
    def git_branch
    def git_tag
    def is_tag_build = false
    def is_pr_build = false
    def docker_image_tag

    def docker_test_image_id_agent
    def docker_test_image_id_coordinator
    def docker_image_id_agent
    def docker_image_id_coordinator

    stage('Checkout') {
        def scmVars = checkout scm
        env.GIT_COMMIT = scmVars.GIT_COMMIT
        env.GIT_BRANCH = scmVars.GIT_BRANCH
        git_commit = scmVars.GIT_COMMIT
        git_branch = scmVars.GIT_BRANCH
        is_tag_build = gitUtils.isTagBuild(git_branch)
        is_pr_build = gitUtils.isPRBuild(git_branch)

        if(is_tag_build) {
            git_tag = scmVars.GIT_BRANCH
            docker_image_tag = git_tag
        } else {
            docker_image_tag = git_commit
        }
    }

    stage('Test Preparation') {
        container('docker') {
            sh "docker version"

            dockerUtils.buildImage("${docker_repo_agent}:${docker_image_tag}-test", dockerfile_test_agent)
            docker_test_image_id_agent = dockerUtils.getImageId(docker_repo_agent, "${docker_image_tag}-test")

            //dockerUtils.buildImage("${docker_repo_coordinator}:${docker_image_tag}-test", dockerfile_test_coordinator)
            //docker_test_image_id_coordinator = dockerUtils.getImageId(docker_repo_coordinator, "${docker_image_tag}-test")
        }
    }

    parallel(
        lint: {
            stage('Test - Linting') {
                container('docker') {
                    dockerUtils.runShellCommand(docker_test_image_id_agent, 'go get -u github.com/golang/lint/golint && PATH=$PATH:/gocode/bin && make lint')
                }
            }
        },
        test: {
            stage('Test - Testing') {
                container('docker') {
                    dockerUtils.runShellCommand(docker_test_image_id_agent, 'make test')
                }
            }
        },
        vet: {
            stage('Test - Vet') {
                container('docker') {
                    dockerUtils.runShellCommand(docker_test_image_id_agent, 'make vet')
                }
            }
        },
        format: {
            stage('Test - Formating') {
                container('docker') {
                    dockerUtils.runShellCommand(docker_test_image_id_agent, '! gofmt -d -s internal pkg cmd 2>&1 | read')
                }
            }
        }
    )

    if(!is_pr_build && git_branch == deploy_branch) {
        stage('Publish Preparation') {
            container('docker') {
                dockerUtils.buildImage("${docker_repo_agent}:${docker_image_tag}-tmp", dockerfile_agent)
                docker_image_id_agent = dockerUtils.getImageId(docker_repo_agent, "${docker_image_tag}-tmp")

                dir('agent-scratch') {
                    // build and copy files from agent-scratch container
                    def agent_container = "extract-agent"
                    dockerUtils.createContainer(agent_container, "${docker_repo_agent}:${docker_image_tag}-tmp")
                    dockerUtils.copyFromContainer(agent_container, "/scripts/containership_login.sh", "containership-login.sh")
                    dockerUtils.copyFromContainer(agent_container, "/etc/ssl/certs/ca-certificates.crt", "ca-certificates.crt")
                    dockerUtils.copyFromContainer(agent_container, "/app/agent", "agent")
                    dockerUtils.removeContainer(agent_container)

                    // create minimal dockerfile
                    sh 'echo "FROM scratch" >> Dockerfile.agent-scratch'
                    sh 'echo "ADD ./containership-login.sh /scripts" >> Dockerfile.agent-scratch'
                    sh 'echo "ADD ./ca-certificates.crt /etc/ssl/certs/ca-certificates.crt" >> Dockerfile.agent-scratch'
                    sh 'echo "ADD ./agent ." >> Dockerfile.agent-scratch'
                    sh 'echo "CMD [\\"./agent\\", \\"-logtostderr=true\\"]" >> Dockerfile.agent-scratch'

                    dockerUtils.buildImage("${docker_repo_agent}:${docker_image_tag}", "./Dockerfile.agent-scratch")
                    docker_image_id_agent = dockerUtils.getImageId(docker_repo_agent, "${docker_image_tag}")
                }

                dockerUtils.buildImage("${docker_repo_coordinator}:${docker_image_tag}-tmp", dockerfile_coordinator)
                docker_image_id_agent = dockerUtils.getImageId(docker_repo_agent, "${docker_image_tag}-tmp")

                dir('coordinator-scratch') {
                    // build and copy files from coordinator-scratch container
                    def coordinator_container = "extract-coordinator"
                    dockerUtils.createContainer(coordinator_container, "${docker_repo_coordinator}:${docker_image_tag}-tmp")
                    dockerUtils.copyFromContainer(coordinator_container, "/etc/ssl/certs/ca-certificates.crt", "ca-certificates.crt")
                    dockerUtils.copyFromContainer(coordinator_container, "/app/coordinator", "coordinator")
                    dockerUtils.copyFromContainer(coordinator_container, "/kubectl", "kubectl")
                    dockerUtils.removeContainer(coordinator_container)

                    // create minimal dockerfile
                    sh 'echo "FROM scratch" >> Dockerfile.coordinator-scratch'
                    sh 'echo "ADD ./coordinator ." >> Dockerfile.coordinator-scratch'
                    sh 'echo "ADD ./kubectl /etc/kubectl/kubectl" >> Dockerfile.coordinator-scratch'
                    sh 'echo "ADD ./ca-certificates.crt /etc/ssl/certs/ca-certificates.crt" >> Dockerfile.coordinator-scratch'
                    sh 'echo "ENV KUBECTL_PATH=/etc/kubectl/kubectl" >> Dockerfile.coordinator-scratch'
                    sh 'echo "CMD [\\"./coordinator\\", \\"-logtostderr=true\\"]" >> Dockerfile.coordinator-scratch'

                    dockerUtils.buildImage("${docker_repo_coordinator}:${docker_image_tag}", "./Dockerfile.coordinator-scratch")
                    docker_image_id_coordinator = dockerUtils.getImageId(docker_repo_coordinator, "${docker_image_tag}")
                }
            }
        }

        stage('Publish - Docker') {
            container('docker') {
                dockerUtils.login(pipelineUtils.getDockerCredentialId())
                dockerUtils.push(docker_repo_agent, docker_image_tag)
                dockerUtils.push(docker_repo_coordinator, docker_image_tag)
            }
        }
    }

    stage('Cleanup') {
        container('docker') {
            dockerUtils.cleanup()
        }
    }
}
