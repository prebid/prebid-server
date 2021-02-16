pipeline {
    options {
       buildDiscarder(logRotator(numToKeepStr: '10'))
    }
    agent any
    stages {
        stage('Build image') {
            steps {
                sh "docker build -t prebid-server ."
            }
        }
        stage('Push image') {
            when { expression { env.TAG_NAME != null }}
            steps {
                sh "docker build -t prebid-server ."
                sh "docker tag prebid-server:latest docker.ayl.io/ayl/prebid-server:${env.TAG_NAME}"
                sh "docker push docker.ayl.io/ayl/tag:${env.TAG_NAME}"
            }
        }
    }
}