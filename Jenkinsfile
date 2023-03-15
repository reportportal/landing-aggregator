node {
  load "$JENKINS_HOME/jobvars.env"
  stage 'Checkout'

  checkout scm

  stage('Make') {
    sh 'make checkstyle test build_docker'
  }

  stage('Create Docker Image') {
    sh "docker build -t ${AWS_URI}/landing-aggregator:SNAPSHOT-${BUILD_NUMBER} ."
  }

  stage ('Push to regestry') {
     sh "aws ecr get-login-password --region ${AWS_REGION} | docker login --username AWS --password-stdin ${AWS_URI}"
     sh "docker push ${AWS_URI}/landing-aggregator:SNAPSHOT-${BUILD_NUMBER}"
   }
   
   stage ('Cleanup') {
     sh "docker rmi ${AWS_URI}/landing-aggregator:SNAPSHOT-${BUILD_NUMBER}"
   }
}
