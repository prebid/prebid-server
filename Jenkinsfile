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
        stage('Deploy Docker image') {
            environment {
                MARATHON_URL = "https://deploy-pprod.ayl.io/v2/apps/tag/prebid-server-test"
                JSON = "{\"container\":{\"docker\":{\"image\":\"docker.ayl.io/ayl/prebid-server:${params.tag}\"}}}"
            }
            steps {
                withCredentials([file(credentialsId: 'p12_cert', variable: 'cert')]) {
                    sh '''
                    curl -H "Content-Type: application/json" -XPUT --cert-type P12 --cert $cert $MARATHON_URL -d $JSON -w %{http_code}
                    '''
                }
            }
            post {
                success {
                    slackSend channel: '#test-guillaume', message: "Prebid-server ${params.tag} deployed!\n"
                }
            }
        }
    }
}