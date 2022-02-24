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

//        stage('Build Operator') {
//            new Docker(this).build('cloudogu/k8s-dogu-operator:0.0.0-dev', '-f Dockerfile .')
//        }

        stage('Build Operator Image (Docker)') {
            new Docker(this).build('cloudogu/k8s-dogu-operator:0.0.0-dev', '-f Dockerfile .')
        }

//        new Docker(this).image('cloudogu/buildbaseline:0.1.0')
//                .mountJenkinsUser()
//                .inside("--volume ${WORKSPACE}:/project -w /project") {
//
//                    stage('Build') {
//                        sh "make clean package"
//                    }
//
//                    stage('Unit Test') {
//                        sh "make unit-test"
//                        junit allowEmptyResults: true, testResults: 'target/unit-tests/*-tests.xml'
//                    }
//
//                    stage('Static Analysis') {
//                        def commitSha = sh(returnStdout: true, script: 'git rev-parse HEAD').trim()
//
//                        withCredentials([
//                                [$class: 'UsernamePasswordMultiBinding', credentialsId: 'sonarqube-gh', usernameVariable: 'USERNAME', passwordVariable: 'REVIEWDOG_GITHUB_API_TOKEN']
//                        ]) {
//                            withEnv(["CI_PULL_REQUEST=${env.CHANGE_ID}", "CI_COMMIT=${commitSha}", "CI_REPO_OWNER=cloudogu", "CI_REPO_NAME=${repositoryName}"]) {
//                                sh "make static-analysis-ci"
//                            }
//                        }
//                    }
//                }

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