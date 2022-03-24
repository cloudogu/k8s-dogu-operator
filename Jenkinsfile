#!groovy

@Library(['github.com/cloudogu/dogu-build-lib@v1.6.0', 'github.com/cloudogu/ces-build-lib@1.51.0'])
import com.cloudogu.ces.cesbuildlib.*
import com.cloudogu.ces.dogubuildlib.*

// Creating necessary git objects
git = new Git(this, "cesmarvin")
git.committerName = 'cesmarvin'
git.committerEmail = 'cesmarvin@cloudogu.com'
gitflow = new GitFlow(this, git)
github = new GitHub(this, git)
changelog = new Changelog(this)
Docker docker = new Docker(this)
gpg = new Gpg(this, docker)

// Configuration of repository
repositoryOwner = "cloudogu"
repositoryName = "k8s-dogu-operator"
project = "github.com/${repositoryOwner}/${repositoryName}"

// Configuration of branches
productionReleaseBranch = "main"
developmentBranch = "develop"
currentBranch = "${env.BRANCH_NAME}"

node('docker') {
    timestamps {
        stage('Checkout') {
            checkout scm
            make 'clean'
        }

        stage('Lint') {
            lintDockerfile()
        }

        docker
                .image('golang:1.17.7')
                .mountJenkinsUser()
                .inside("--volume ${WORKSPACE}:/go/src/${project} -w /go/src/${project}")
                        {
                            stage('Build') {
                                withCredentials([usernamePassword(credentialsId: 'cesmarvin',
                                        passwordVariable: 'CES_MARVIN_PASSWORD',
                                        usernameVariable: 'CES_MARVIN_USERNAME')]) {
                                    // .netrc is necessary to access private repos
                                    sh "echo \"machine github.com\n" +
                                            "login ${CES_MARVIN_USERNAME}\n" +
                                            "password ${CES_MARVIN_PASSWORD}\" >> ~/.netrc"
                                }
                                make 'build-controller'
                            }

                            stage('k8s-Integration-Test') {
                                make 'k8s-integration-test'
                            }

                            stage("Review dog analysis") {
                                stageStaticAnalysisReviewDog()
                            }

                            stage('Generate k8s Resources') {
                                make 'k8s-generate'
                                archiveArtifacts 'target/*.yaml'
                            }
                        }

        stage("Lint k8s Resources") {
            stageLintK8SResources()
        }

        stage('SonarQube') {
            stageStaticAnalysisSonarQube()
        }

        K3d k3d = new K3d(this, "${WORKSPACE}/k3d", env.PATH)

        try {
            Makefile makefile = new Makefile(this)
            String controllerVersion = makefile.getVersion()

            stage('Set up k3d cluster') {
                k3d.startK3d()
            }

            def imageName
            stage('Build & Push Image') {
                imageName=k3d.buildAndPushToLocalRegistry("cloudogu/${repositoryName}", controllerVersion)
            }

            GString sourceDeploymentYaml="target/${repositoryName}_${controllerVersion}.yaml"
            stage('Update development resources') {
                docker.image('mikefarah/yq:4.22.1')
                        .mountJenkinsUser()
                        .inside("--volume ${WORKSPACE}:/workdir -w /workdir") {
                            sh "yq -i '(select(.kind == \"Deployment\").spec.template.spec.containers[]|select(.name == \"manager\")).image=\"${imageName}\"' ${sourceDeploymentYaml}"
                        }
            }

            stage('Deploy Manager') {
                k3d.kubectl("apply -f ${sourceDeploymentYaml}")
            }

            stage('Wait for Ready Rollout') {
                k3d.kubectl("--namespace default wait --for=condition=Ready pods --all")
            }

            stageAutomaticRelease()
        } finally {
            stage('Remove k3d cluster') {
                k3d.deleteK3d()
            }
        }
    }
}

void gitWithCredentials(String command) {
    withCredentials([usernamePassword(credentialsId: 'cesmarvin', usernameVariable: 'GIT_AUTH_USR', passwordVariable: 'GIT_AUTH_PSW')]) {
        sh(
                script: "git -c credential.helper=\"!f() { echo username='\$GIT_AUTH_USR'; echo password='\$GIT_AUTH_PSW'; }; f\" " + command,
                returnStdout: true
        )
    }
}

void stageLintK8SResources() {
    String kubevalImage = "cytopia/kubeval:0.13"
    Makefile makefile = new Makefile(this)
    String controllerVersion = makefile.getVersion()

    docker
            .image(kubevalImage)
            .inside("-v ${WORKSPACE}/target:/data -t --entrypoint=")
                    {
                        sh "kubeval /data/${repositoryName}_${controllerVersion}.yaml --ignore-missing-schemas"
                    }
}

void stageStaticAnalysisReviewDog() {
    def commitSha = sh(returnStdout: true, script: 'git rev-parse HEAD').trim()

    withCredentials([[$class: 'UsernamePasswordMultiBinding', credentialsId: 'sonarqube-gh', usernameVariable: 'USERNAME', passwordVariable: 'REVIEWDOG_GITHUB_API_TOKEN']]) {
        withEnv(["CI_PULL_REQUEST=${env.CHANGE_ID}", "CI_COMMIT=${commitSha}", "CI_REPO_OWNER=${repositoryOwner}", "CI_REPO_NAME=${repositoryName}"]) {
            make 'static-analysis-ci'
        }
    }
}

void stageStaticAnalysisSonarQube() {
    def scannerHome = tool name: 'sonar-scanner', type: 'hudson.plugins.sonar.SonarRunnerInstallation'
    withSonarQubeEnv {
        sh "git config 'remote.origin.fetch' '+refs/heads/*:refs/remotes/origin/*'"
        gitWithCredentials("fetch --all")

        if (currentBranch == productionReleaseBranch) {
            echo "This branch has been detected as the production branch."
            sh "${scannerHome}/bin/sonar-scanner -Dsonar.branch.name=${env.BRANCH_NAME}"
        } else if (currentBranch == developmentBranch) {
            echo "This branch has been detected as the development branch."
            sh "${scannerHome}/bin/sonar-scanner -Dsonar.branch.name=${env.BRANCH_NAME}"
        } else if (env.CHANGE_TARGET) {
            echo "This branch has been detected as a pull request."
            sh "${scannerHome}/bin/sonar-scanner -Dsonar.pullrequest.key=${env.CHANGE_ID} -Dsonar.pullrequest.branch=${env.CHANGE_BRANCH} -Dsonar.pullrequest.base=${developmentBranch}"
        } else if (currentBranch.startsWith("feature/")) {
            echo "This branch has been detected as a feature branch."
            sh "${scannerHome}/bin/sonar-scanner -Dsonar.branch.name=${env.BRANCH_NAME}"
        } else {
            echo "This branch has been detected as a miscellaneous branch."
            sh "${scannerHome}/bin/sonar-scanner -Dsonar.branch.name=${env.BRANCH_NAME} "
        }
    }
    timeout(time: 2, unit: 'MINUTES') { // Needed when there is no webhook for example
        def qGate = waitForQualityGate()
        if (qGate.status != 'OK') {
            unstable("Pipeline unstable due to SonarQube quality gate failure")
        }
    }
}

void stageAutomaticRelease() {
    if (gitflow.isReleaseBranch()) {
        String releaseVersion = git.getSimpleBranchName()
        String dockerReleaseVersion = releaseVersion.split("v")[1]

        stage('Build & Push Image') {
            withCredentials([usernamePassword(credentialsId: 'cesmarvin',
                    passwordVariable: 'CES_MARVIN_PASSWORD',
                    usernameVariable: 'CES_MARVIN_USERNAME')]) {
                // .netrc is necessary to access private repos
                sh "echo \"machine github.com\n" +
                        "login ${CES_MARVIN_USERNAME}\n" +
                        "password ${CES_MARVIN_PASSWORD}\" >> ~/.netrc"
            }
            def dockerImage = docker.build("cloudogu/${repositoryName}:${dockerReleaseVersion}")
            sh "rm ~/.netrc"
            docker.withRegistry('https://registry.hub.docker.com/', 'dockerHubCredentials') {
                dockerImage.push("${dockerReleaseVersion}")
            }
        }

        stage('Finish Release') {
            gitflow.finishRelease(releaseVersion, productionReleaseBranch)
        }

        stage('Sign after Release') {
            gpg.createSignature()
        }

        stage('Add Github-Release') {
            Makefile makefile = new Makefile(this)
            String controllerVersion = makefile.getVersion()
            GString targetOperatorResourceYaml = "target/${repositoryName}_${controllerVersion}.yaml"
            releaseId = github.createReleaseWithChangelog(releaseVersion, changelog, productionReleaseBranch)
            github.addReleaseAsset("${releaseId}", "${targetOperatorResourceYaml}")
            github.addReleaseAsset("${releaseId}", "${targetOperatorResourceYaml}.sha256sum")
            github.addReleaseAsset("${releaseId}", "${targetOperatorResourceYaml}.sha256sum.asc")
        }
    }
}

void make(String makeArgs) {
    sh "make ${makeArgs}"
}