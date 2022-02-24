#!groovy

@Library(['github.com/cloudogu/dogu-build-lib@v1.4.1', 'github.com/cloudogu/ces-build-lib@v1.48.0'])
import com.cloudogu.ces.cesbuildlib.*
import com.cloudogu.ces.dogubuildlib.*

// Creating necessary git objects
Git git = new Git(this, "cesmarvin")
git.committerName = 'cesmarvin'
git.committerEmail = 'cesmarvin@cloudogu.com'
GitFlow gitflow = new GitFlow(this, git)
GitHub github = new GitHub(this, git)
Changelog changelog = new Changelog(this)

// Configuration of repository
String repositoryOwner = 'cloudogu'
String repositoryName = "k8s-dogu-operator"
String project = "github.com/${repositoryOwner}/${repositoryName}"

// Configuration of branches
String productionReleaseBranch = "main"
String developmentBranch = "develop"
String currentBranch = "${env.BRANCH_NAME}"

node('docker') {
    timestamps {
        stage('Checkout') {
            checkout scm
        }

        new Docker(this)
                .image('golang:1.17.7')
                .inside("--volume ${WORKSPACE}:/go/src/${project} -w /go/src/${project}")
                        {
                            stage('Build') {
                                sh "make build"
                            }

                            stage('Test') {
                                sh "make test"
                            }
                        }

        stage('Build Operator Image (Docker)') {
            new Docker(this).build('cloudogu/k8s-dogu-operator:0.0.0-dev', '-f Dockerfile .')
        }

        stage('SonarQube') {
            def scannerHome = tool name: 'sonar-scanner', type: 'hudson.plugins.sonar.SonarRunnerInstallation'
            withSonarQubeEnv {
                sh "git config 'remote.origin.fetch' '+refs/heads/*:refs/remotes/origin/*'"
                gitWithCredentials("fetch --all")

                if (currentBranch == productionReleaseBranch) {
                    echo "This currentBranch has been detected as the production branch."
                    sh "${scannerHome}/bin/sonar-scanner -Dsonar.currentBranch.name=${env.BRANCH_NAME}"
                } else if (currentBranch == developmentBranch) {
                    echo "This currentBranch has been detected as the development branch."
                    sh "${scannerHome}/bin/sonar-scanner -Dsonar.currentBranch.name=${env.BRANCH_NAME}"
                } else if (env.CHANGE_TARGET) {
                    echo "This currentBranch has been detected as a pull request."
                    sh "${scannerHome}/bin/sonar-scanner -Dsonar.pullrequest.key=${env.CHANGE_ID} -Dsonar.pullrequest.currentBranch=${env.CHANGE_BRANCH} -Dsonar.pullrequest.base=${developmentBranch}"
                } else if (currentBranch.startsWith("feature/")) {
                    echo "This currentBranch has been detected as a feature branch."
                    sh "${scannerHome}/bin/sonar-scanner -Dsonar.currentBranch.name=${env.BRANCH_NAME}"
                } else {
                    echo "This currentBranch has been detected as a miscellaneous currentBranch."
                    sh "${scannerHome}/bin/sonar-scanner -Dsonar.currentBranch.name=${env.BRANCH_NAME} "
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
}

void gitWithCredentials(String command) {
    withCredentials([usernamePassword(credentialsId: 'cesmarvin', usernameVariable: 'GIT_AUTH_USR', passwordVariable: 'GIT_AUTH_PSW')]) {
        sh(
                script: "git -c credential.helper=\"!f() { echo username='\$GIT_AUTH_USR'; echo password='\$GIT_AUTH_PSW'; }; f\" " + command,
                returnStdout: true
        )
    }
}