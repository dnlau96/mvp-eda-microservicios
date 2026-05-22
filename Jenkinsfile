pipeline {
  agent any

  environment {
    COMPOSE_PROJECT_NAME = 'mvp_eda'
    SMOKE_BASE_URL = 'host.docker.internal'
  }

  stages {
    stage('Checkout') {
      steps {
        echo 'Descargando codigo desde GitHub...'
        checkout scm
      }
    }

    stage('Validate') {
      steps {
        echo 'Validando estructura de Docker Compose...'
        sh 'docker compose config --quiet'
      }
    }

    stage('Lint/Test') {
      steps {
        echo 'Validando archivos minimos de cada microservicio...'
        sh 'test -f services/asistencia/app.py'
        sh 'test -f services/pagos/index.js'
        sh 'test -f services/panel/main.go'
        echo 'Pruebas ligeras OK. La compilacion real ocurre durante docker compose build.'
      }
    }

    stage('Build') {
      steps {
        echo 'Construyendo imagenes de microservicios...'
        sh 'docker compose build asistencia-service pagos-service panel-service'
      }
    }

    stage('Deploy') {
      steps {
        echo 'Levantando stack completo con Docker Compose...'
        sh 'docker compose up -d rabbitmq postgres mongo mysql keycloak asistencia-service pagos-service panel-service'
      }
    }

    stage('Smoke Test') {
      steps {
        echo 'Verificando endpoints principales...'
        sh 'curl -fsS http://${SMOKE_BASE_URL}:8001 > /dev/null'
        sh 'curl -fsS http://${SMOKE_BASE_URL}:8002 > /dev/null'
        sh 'curl -fsS http://${SMOKE_BASE_URL}:8003 > /dev/null'
      }
    }
  }

  post {
    success {
      echo 'CI/CD finalizado correctamente. MVP desplegado.'
    }
    failure {
      echo 'El pipeline fallo. Revisar logs de Jenkins.'
    }
  }
}
