#!groovy

@Library(['github.com/cloudogu/dogu-build-lib@v1.6.0', 'github.com/cloudogu/ces-build-lib@3a7f24db'])
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
goVersion = "1.18"

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
        }

        stage('Regenerate resources for release') {
            new Docker(this)
                    .image("golang:${goVersion}")
                    .mountJenkinsUser()
                    .inside("--volume ${WORKSPACE}:/go/src/${project} -w /go/src/${project}")
                            {
                                withCredentials([usernamePassword(credentialsId: 'cesmarvin',
                                                                        passwordVariable: 'CES_MARVIN_PASSWORD',
                                                                        usernameVariable: 'CES_MARVIN_USERNAME')]) {
                                    // .netrc is necessary to access private repos
                                    sh "echo \"machine github.com\n" +
                                            "login ${CES_MARVIN_USERNAME}\n" +
                                            "password ${CES_MARVIN_PASSWORD}\" >> ~/.netrc"
                                }
                                make 'k8s-create-temporary-resource'
                            }
        }

        Makefile makefile = new Makefile(this)
        String controllerVersion = makefile.getVersion()
        GString targetOperatorResourceYaml = "target/${repositoryName}_${controllerVersion}.yaml"

        DoguRegistry registry = new DoguRegistry(this, "https://staging-dogu.cloudogu.com")
        registry.pushK8sYaml(targetOperatorResourceYaml, repositoryName, "k8s", "${controllerVersion}")
    }
}

void make(String makeArgs) {
    sh "make ${makeArgs}"
}