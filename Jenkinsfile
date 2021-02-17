pipeline {
    options {
       buildDiscarder(logRotator(numToKeepStr: '10'))
    }
    agent any

    parameters {
        string(description: 'Set an arbitrary build tag name. Please reference prebid-server version.', name: 'tag')
    }

    stages {
        stage('Build image') {
            steps {
                sh "docker build -t prebid-server ."
            }
        }
        stage('Push image') {
            steps {
                sh "docker tag prebid-server:latest docker.ayl.io/ayl/prebid-server:${params.tag}"
                sh "docker push docker.ayl.io/ayl/prebid-server:${params.tag}"
            }
        }
    }
}