node {
    stage 'Checkout'

    checkout scm

     stage('Create Docker Image') {
          docker.build("reportportal/landing-aggregator:latest")
     }

     stage ('Run Application') {
           // Run application using Docker image
           sh "docker-compose -p aggregator up -d --force-recreate"
     }
}