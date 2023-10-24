#!groovy

pipeline {
    environment {
        CI = "false"
        MY_VERSION = sh(
                script: 'if [[ $BRANCH_NAME =~ "-alkimi" ]]; then echo "${BRANCH_NAME}"; else echo "0.0.${BUILD_ID}-${BRANCH_NAME}-SNAPSHOT"; fi',
                returnStdout: true
        ).trim()
        MY_ENV = sh(
                script: 'if [[ $BRANCH_NAME =~ "REL_V" ]]; then echo "prod"; elif [[ $BRANCH_NAME =~ "sprint-" ]]; then echo qa; else echo "dev"; fi',
                returnStdout: true
        ).trim()
        DO_API_TOKEN = vault path: 'jenkins/digitalocean', key: 'ro_token'
    }
    options {
        disableConcurrentBuilds()
    }
    agent any
    stages {
        stage('Tests') {
           steps {
               script {
                  sh "./validate.sh"
               }
           }
        }
	    stage('Build') {
            steps {
                script {
                    sh "echo ${BRANCH_NAME} ${GIT_BRANCH} ${GIT_COMMIT} ${MY_VERSION} ${MY_ENV}"
			        sh "go build ."
                }
            }
        }
        // stage('Deploy to dev') {
        //     when {
        //         branch "master_asterio"
        //     }
        //     steps {
        //         dir('ansible') {
        //             git branch: 'master', url: "git@github.com:Alkimi-Exchange/alkimi-ansible.git", credentialsId: 'ssh-alkimi-ansible'
        //         }
        //         sh "cd ./ansible && ansible-playbook ./apps/dev/prebid-server.yml --extra-vars='artifactPath=${env.WORKSPACE}/target/prebid-server.jar configPath=${env.WORKSPACE}/config'"
        //     }
        // }
        // stage('Build and push docker images') {
        //     //when { tag "REL_V*" }
        //     steps {
        //         script {
        //             if (env.BRANCH_NAME =~ "\\d+\\.\\d+\\.\\d+-alkimi") {
        //                 docker.withRegistry('https://685748726849.dkr.ecr.eu-west-2.amazonaws.com','ecr:eu-west-2:jenkins_ecr') {
        //                     def dockerImage = docker.build("alkimi/prebid-server:${MY_VERSION}", "--build-arg APP_NAME=prebid-server -f docker/Dockerfile ${WORKSPACE}")
        //                     dockerImage.push()
        //                     dockerImage.push('latest')
        //                 }
        //             }
        //         }
        //     }
        // }
    }
}
