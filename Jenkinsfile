#!groovy

@Library(['github.com/cloudogu/dogu-build-lib@v1.4.1', 'github.com/cloudogu/ces-build-lib@v1.48.0'])
import com.cloudogu.ces.cesbuildlib.*
import com.cloudogu.ces.dogubuildlib.*

// Creating necessary git objects
git = new Git(this, "cesmarvin")
git.committerName = 'cesmarvin'
git.committerEmail = 'cesmarvin@cloudogu.com'
gitflow = new GitFlow(this, git)
github = new GitHub(this, git)
changelog = new Changelog(this)

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
        stageCheckoutProject()
//      stageLintDockerfile()

        new Docker(this)
                .image('golang:1.17.7')
                .mountJenkinsUser()
                .inside("--volume ${WORKSPACE}:/go/src/${project} -w /go/src/${project}")
                        {
                            stageBuildController()
                            stageK8SIntegrationTest()
                        }

        stageStaticAnalysis()
        stageAutomaticRelease()

        stage('Build Image') {
            make 'docker-build'
        }

        stage('Generate k8s Resources') {
            make 'k8s-generate'
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

void stageCheckoutProject() {
    stage('Checkout') {
        checkout scm
    }
}

void stageLintDockerfile() {
    stage('Lint') {
        lintDockerfile()
    }
}

void stageBuildController() {
    stage('Build') {
        sh "make build"
    }
}

void stageK8SIntegrationTest() {
    stage('k8s-Integration-Test') {
        sh "make k8s-integration-test"
    }
}

void stageStaticAnalysis() {
    stage('SonarQube') {
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
}

void stageAutomaticRelease() {
    if (gitflow.isReleaseBranch()) {
        String releaseVersion = git.getSimpleBranchName()

        stage('Build Image') {
            make 'docker-build'
        }

        stage('Generate k8s Resources') {
            make 'k8s-generate'
        }

        stage('Finish Release') {
            gitflow.finishRelease(releaseVersion)
        }

        withBuildDependencies {
            stage('Build after Release') {
                git.checkout(releaseVersion)
                make 'clean build checksum'
            }
        }

        stage('Sign after Release'){
            gpg.createSignature()
        }

        stage('Add Github-Release') {
            releaseId=github.createReleaseWithChangelog(releaseVersion, changelog)
            github.addReleaseAsset("${releaseId}", "target/k8s-dogu-operator.sha256sum")
            github.addReleaseAsset("${releaseId}", "target/k8s-dogu-operator.sha256sum.asc")
            github.addReleaseAsset("${releaseId}", "dist/latest.yaml")
        }
    }
}