public boolean isTagBuild(commit) {
    def ref = sh(returnStdout: true, script: "git show-ref --tags | awk '\$1==\"${commit}\" { print \$2 }'").trim()
    return ref.contains("refs/tags/")
}

public boolean isPRBuild(branch) {
    return branch.contains("PR-");
}

public void jenkinsTemplate(body) {
    podTemplate(
            label: 'k8s-pod',
            containers: [
                containerTemplate(name: 'docker', image: 'docker', command: 'cat', ttyEnabled: true),
                containerTemplate(name: 'kubectl', image: 'lachlanevenson/k8s-kubectl:v1.9.2', command: 'cat', ttyEnabled: true)
            ],
            volumes: [hostPathVolume(hostPath: '/var/run/docker.sock', mountPath: '/var/run/docker.sock')]) {
        body()
    }
}

public String getImageName() {
    return "${env.CS_IMAGE_ORG}/${env.CS_IMAGE_NAME}:${env.CS_IMAGE_TAG}"
}

public String getImageId(name, tag) {
    def match = "\$1 == \"${name}\" && \$2 == \"${tag}\""
    return sh (returnStdout: true, script: "docker images | awk '${match} { print \$3 }'").trim()
}

public void runShellCommand(imageId, cmd) {
    return runCommand(imageId, 'sh', "-c '${cmd}'")
}

public void runCommand(imageId, entrypoint, cmd) {
    def result = sh(script: "docker run --entrypoint=${entrypoint} -i ${imageId} ${cmd}", returnStatus: true)

    if(result != 0) {
        throw new Exception("Failed to run cmd[${cmd}] on image[${imageName}]")
    }
}

public void dockerLogin(userPassCredentialId) {
    withCredentials([
        usernamePassword(credentialsId: userPassCredentialId, usernameVariable: 'DOCKER_USER', passwordVariable: 'DOCKER_PASS')
    ]) {
        sh(returnStdout: true, script: "docker login -u ${env.DOCKER_USER} -p ${env.DOCKER_PASS}")
    }
}

public void dockerPush(name, tag) {
    sh(returnStatus: true, script: "docker push ${name}:${tag}")
}

public void dockerTag(name, original_tag, new_tag) {
    sh(returnStatus: true, script: "docker tag ${name}:${orginal_tag} ${name}:${new_tag}")
}

public void npmLogin(npmAuthTokenId) {
    withCredentials([
        string(credentialsId: npmAuthTokenId, variable: 'NPM_TOKEN')
    ]) {
        sh(returnStdout: true, script: "echo //registry.npmjs.org/:_authToken=${env.NPM_TOKEN} > ./.npmrc")
    }
}

public void getPackageVersion(path) {
    return sh(returnStdout: true, script: "cat ${path} | grep version | head -1 | awk -F: '{print \$2 }' | sed 's/[\",]//g'");
}

jenkinsTemplate {
    node('k8s-pod') {
        def docker_org = 'ntate22'
        def docker_name_agent = 'cloud-agent'
        def docker_name_coordinator = 'cloud-coordinator'

        def dockerfile_test_agent = 'Dockerfile'
        def dockerfile_test_coordinator = 'Dockerfile'

        def dockerfile_agent = 'Dockerfile.agent'
        def dockerfile_coordinator = 'Dockerfile.coordinator'

        def docker_repo_agent = "${docker_org}/${docker_name_agent}"
        def docker_repo_coordinator = "${docker_org}/${docker_name_coordinator}"

        // will be set in checkout stage
        def git_commit
        def git_branch
        def git_tag
        def is_tag_build = false
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
            is_tag_build = isTagBuild(scmVars.GIT_COMMIT)

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

                sh (returnStatus: true, script: "docker build -t ${docker_repo_agent}:${docker_image_tag}-test -f ${dockerfile_test_agent} .")
                docker_test_image_id_agent = getImageId(docker_repo_agent, "${docker_image_tag}-test")

                //sh (returnStatus: true, script: "docker build -t ${docker_repo_coordinator}:${docker_image_tag}-test -f ${dockerfile_test_coordinator} .")
                //docker_test_image_id_coordinator = getImageId(docker_repo_coordinator, "${docker_image_tag}-test")
            }
        }

        parallel(
            lint: {
                stage('Test - Linting') {
                    container('docker') {
                        runShellCommand(docker_test_image_id_agent, 'go get -u github.com/golang/lint/golint && PATH=$PATH:/gocode/bin && make lint')
                    }
                }
            },
            test: {
                stage('Test - Testing') {
                    container('docker') {
                        runShellCommand(docker_test_image_id_agent, 'make test')
                    }
                }
            },
            vet: {
                stage('Test - Vet') {
                    container('docker') {
                        runShellCommand(docker_test_image_id_agent, 'make vet')
                    }
                }
            },
            format: {
                stage('Test - Formating') {
                    container('docker') {
                        runShellCommand(docker_test_image_id_agent, '! gofmt -d -s internal pkg cmd 2>&1 | read')
                    }
                }
            }
        )

        if(!isPRBuild(git_branch)) {
            stage('Publish Preparation') {
                container('docker') {
                    sh (returnStatus: true, script: "docker build -t ${docker_repo_agent}:${docker_image_tag}-tmp -f ${dockerfile_agent} .")
                    docker_image_id_agent = getImageId(docker_repo_agent, "${docker_image_tag}-tmp")

                    dir('agent-scratch') {
                        // build and copy files from agent-scratch container
                        def agent_container = "extract-agent"
                        sh(returnStdout: true, script: "docker container create --name ${agent_container} ${docker_repo_agent}:${docker_image_tag}-tmp").trim()
                        sh(returnStatus: true, script: "docker container cp ${agent_container}:/scripts/containership_login.sh containership-login.sh")
                        sh(returnStatus: true, script: "docker container cp ${agent_container}:/etc/ssl/certs/ca-certificates.crt ca-certificates.crt")
                        sh(returnStatus: true, script: "docker container cp ${agent_container}:/app/agent agent")
                        sh(returnStatus: true, script: "docker rm ${agent_container}")

                        // create minimal dockerfile
                        sh 'echo "FROM scratch" >> Dockerfile.agent-scratch'
                        sh 'echo "ADD ./containership-login.sh /scripts" >> Dockerfile.agent-scratch'
                        sh 'echo "ADD ./ca-certificates.crt /etc/ssl/certs/ca-certificates.crt" >> Dockerfile.agent-scratch'
                        sh 'echo "ADD ./agent ." >> Dockerfile.agent-scratch'
                        sh 'echo "CMD [\"./agent\", \"-logtostderr=true\"]" >> Dockerfile.agent-scratch'

                        sh (returnStatus: true, script: "docker build -t ${docker_repo_agent}:${docker_image_tag} -f ./Dockerfile.agent-scratch .")
                        docker_image_id_agent = getImageId(docker_repo_agent, "${docker_image_tag}")
                    }

                    sh (returnStatus: true, script: "docker build -t ${docker_repo_coordinator}:${docker_image_tag}-tmp -f ${dockerfile_coordinator} .")
                    docker_image_id_coordinator = getImageId(docker_repo_coordinator, "${docker_image_tag}-tmp")

                    dir('coordinator-scratch') {
                        // build and copy files from coordinator-scratch container
                        def coordinator_container = "extract-coordinator"
                        sh(returnStdout: true, script: "docker container create --name ${coordinator_container} ${docker_repo_coordinator}:${docker_image_tag}-tmp").trim()
                        sh(returnStatus: true, script: "docker container cp ${coordinator_container}:/etc/ssl/certs/ca-certificates.crt ca-certificates.crt")
                        sh(returnStatus: true, script: "docker container cp ${coordinator_container}:/app/coordinator coordinator")
                        sh(returnStatus: true, script: "docker container cp ${coordinator_container}:/kubectl kubectl")
                        sh(returnStatus: true, script: "docker rm ${coordinator_container}")

                        // create minimal dockerfile
                        sh 'echo "FROM scratch" >> Dockerfile.coordinator-scratch'
                        sh 'echo "ADD ./coordinator ." >> Dockerfile.coordinator-scratch'
                        sh 'echo "ADD ./kubectl /etc/kubectl/kubectl" >> Dockerfile.coordinator-scratch'
                        sh 'echo "ADD ./ca-certificates.crt /etc/ssl/certs/ca-certificates.crt" >> Dockerfile.coordinator-scratch'
                        sh 'echo "ENV KUBECTL_PATH=/etc/kubectl/kubectl" >> Dockerfile.coordinator-scratch'
                        sh 'echo "CMD [\"./coordinator\", \"-logtostderr=true\"]" >> Dockerfile.coordinator-scratch'

                        sh (returnStatus: true, script: "docker build -t ${docker_repo_coordinator}:${docker_image_tag} -f ./Dockerfile.coordinator-scratch .")
                        docker_image_id_coordinator = getImageId(docker_repo_coordinator, "${docker_image_tag}")
                    }
                }
            }

            stage('Publish - Docker') {
                if(!isPRBuild(git_branch)) {
                    container('docker') {
                        dockerLogin('docker-cred-ntate22')
                        dockerPush(docker_repo_agent, docker_image_tag)
                        dockerPush(docker_repo_coordinator, docker_image_tag)
                    }
                }
            }

            stage('Deploy - Kubernetes') {
                echo "We need to support deploying to kubernetes clusters"

                container('kubectl') {
                    sh "kubectl config get-contexts"
                    sh "kubectl get pods"

                    withCredentials([
                        file(credentialsId: 'kube_config_gke_jenkins', variable: 'KUBE_CONFIG')
                    ]) {
                        sh "kubectl --kubeconfig=$KUBE_CONFIG config view"
                        sh "kubectl --kubeconfig=$KUBE_CONFIG config get-contexts"
                        sh "kubectl --kubeconfig=$KUBE_CONFIG get pods"
                    }
                }
            }
        }

        stage('Cleanup') {
            sh 'printenv'

            container('docker') {
                sh 'docker images'
            }
        }
    }
}
