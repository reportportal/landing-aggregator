node {
    stage 'Checkout'

    checkout scm

     stage('Compile') {
          sh "make checkstyle test build_docker"
     }

     stage('Create Docker Image') {
          docker.build("reportportal/landing-aggregator:latest")
     }

     stage ('Run Application') {
           // Run application using Docker image
           sh "docker-compose -p aggregator up -d --force-recreate"
     }
}