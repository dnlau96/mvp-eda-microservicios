pipeline {
  agent any

  stages {
    stage('Checkout') {
      steps {
        echo 'Descargando codigo desde el repositorio...'
        checkout scm
      }
    }

    stage('Lint/Test') {
      steps {
        echo 'Simulando lint y pruebas del servicio de asistencia...'
        sh 'python --version || true'
      }
    }

    stage('Build & Deploy') {
      steps {
        echo 'Construyendo y reiniciando asistencia-service...'
        sh 'docker compose build asistencia-service'
        sh 'docker compose restart asistencia-service'
      }
    }
  }
}
