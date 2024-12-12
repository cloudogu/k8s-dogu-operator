#!groovy

@Library('github.com/cloudogu/ces-build-lib@3.1.0')
import com.cloudogu.ces.cesbuildlib.*

// Creating necessary git objects
git = new Git(this, "cesmarvin")
git.committerName = 'cesmarvin'
git.committerEmail = 'cesmarvin@cloudogu.com'
gitflow = new GitFlow(this, git)
github = new GitHub(this, git)
changelog = new Changelog(this)
Docker docker = new Docker(this)
gpg = new Gpg(this, docker)
goVersion = "1.23.2"
makefile = new Makefile(this)

// Configuration of repository
repositoryOwner = "cloudogu"
repositoryName = "k8s-dogu-operator"
project = "github.com/${repositoryOwner}/${repositoryName}"
registry = "registry.cloudogu.com"
registry_namespace = "k8s"

// Configuration of branches
productionReleaseBranch = "main"
developmentBranch = "develop"
currentBranch = "${env.BRANCH_NAME}"
k8sTargetDir = "target/k8s"
helmChartDir = "${k8sTargetDir}/helm"
helmCRDChartDir = "${k8sTargetDir}/helm-crd"

node('docker') {
    timestamps {
        stage('Checkout') {
            checkout scm
            make 'clean'
        }

        stage('Lint') {
            lintDockerfile()
        }

        stage('Check Markdown Links') {
            Markdown markdown = new Markdown(this, "3.11.0")
            markdown.check()
        }

        new Docker(this)
                .image("golang:${goVersion}")
                .mountJenkinsUser()
                .inside("--volume ${WORKSPACE}:/go/src/${project} -w /go/src/${project}")
                        {
                            stage('Build') {
                                make 'build-controller'
                            }

                            stage("Unit test") {
                                make 'unit-test'
                                junit allowEmptyResults: true, testResults: 'target/unit-tests/*-tests.xml'
                            }

                            stage('k8s-Integration-Test') {
                                make 'k8s-integration-test'
                            }

                            stage("Review dog analysis") {
                                stageStaticAnalysisReviewDog()
                            }

                            stage('Generate k8s Resources') {
                                make 'crd-helm-generate'
                                make 'helm-generate'
                                archiveArtifacts "${k8sTargetDir}/**/*"
                            }

                            stage("Lint helm") {
                                make 'crd-helm-lint'
                                make 'helm-lint'
                            }
                        }

        stage('SonarQube') {
            stageStaticAnalysisSonarQube()
        }


        K3d k3d = new K3d(this, "${WORKSPACE}", "${WORKSPACE}/k3d", env.PATH)

        try {
            String controllerVersion = makefile.getVersion()

            stage('Set up k3d cluster') {
                k3d.startK3d()
            }

            def imageName = ""
            stage('Build & Push Image') {
                imageName = k3d.buildAndPushToLocalRegistry("cloudogu/${repositoryName}", controllerVersion)
            }

            stage('Create initial global config map') {
                k3d.kubectl("--namespace default create configmap global-config --from-literal=config.yaml={}")
            }

            stage('Update development resources') {
                def repository = imageName.substring(0, imageName.lastIndexOf(":"))
                docker.image("golang:${goVersion}")
                        .mountJenkinsUser()
                        .inside("--volume ${WORKSPACE}:/workdir -w /workdir") {
                            sh "STAGE=development IMAGE_DEV=${repository} make helm-values-replace-image-repo"
                        }
            }

            stage('Deploy Manager') {
                k3d.helm("install ${repositoryName}-crd ${helmCRDChartDir}")
                k3d.helm("install ${repositoryName} ${helmChartDir}")
            }

            stage('Wait for Ready Rollout') {
                k3d.kubectl("--namespace default wait --for=condition=Ready pods --all")
            }

            stageAutomaticRelease()
        } catch(Exception e) {
            k3d.collectAndArchiveLogs()
            throw e as java.lang.Throwable
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
        String controllerVersion = makefile.getVersion()
        String releaseVersion = "v${controllerVersion}".toString()

        stage('Build & Push Image') {
            def dockerImage = docker.build("cloudogu/${repositoryName}:${controllerVersion}")
            docker.withRegistry('https://registry.hub.docker.com/', 'dockerHubCredentials') {
                dockerImage.push("${controllerVersion}")
            }
        }

        stage('Finish Release') {
            gitflow.finishRelease(releaseVersion, productionReleaseBranch)
        }

        stage('Sign after Release') {
            gpg.createSignature()
        }

        stage('Push Helm chart to Harbor') {
            new Docker(this)
                .image("golang:${goVersion}")
                .mountJenkinsUser()
                .inside("--volume ${WORKSPACE}:/go/src/${project} -w /go/src/${project}")
                        {
                            make 'helm-package'
                            make 'crd-helm-package'
                            archiveArtifacts "${k8sTargetDir}/**/*"

                            // Push charts
                            withCredentials([usernamePassword(credentialsId: 'harborhelmchartpush', usernameVariable: 'HARBOR_USERNAME', passwordVariable: 'HARBOR_PASSWORD')]) {
                                sh ".bin/helm registry login ${registry} --username '${HARBOR_USERNAME}' --password '${HARBOR_PASSWORD}'"

                                sh ".bin/helm push ${helmChartDir}/${repositoryName}-${controllerVersion}.tgz oci://${registry}/${registry_namespace}/"
                                // Disabled until the CRDs are in their own repo and can be released separately
                                // sh ".bin/helm push ${helmCRDChartDir}/${repositoryName}-crd-${controllerVersion}.tgz oci://${registry}/${registry_namespace}/"
                            }
                        }
        }

        stage('Add Github-Release') {
            releaseId = github.createReleaseWithChangelog(releaseVersion, changelog, productionReleaseBranch)
        }
    }
}

void make(String makeArgs) {
    sh "make ${makeArgs}"
}
