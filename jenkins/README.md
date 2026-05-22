# Jenkins CI/CD

Este proyecto incluye un Jenkins local para ejecutar el `Jenkinsfile` contra Docker Compose.

## Levantar Jenkins

```cmd
docker compose up -d --build jenkins
```

Abrir:

```text
http://localhost:8081
```

El usuario inicial se configura desde el asistente de Jenkins.

## Obtener password inicial

```cmd
docker compose exec jenkins cat /var/jenkins_home/secrets/initialAdminPassword
```

## Plugins recomendados

Durante el asistente selecciona "Install suggested plugins".

Para GitHub puedes instalar o verificar:

```text
Git
Pipeline
GitHub
Docker Pipeline
```

## Crear pipeline

1. New Item.
2. Nombre: `mvp-eda-ci-cd`.
3. Tipo: `Pipeline`.
4. En "Pipeline", seleccionar `Pipeline script from SCM`.
5. SCM: `Git`.
6. Repository URL:

```text
https://github.com/dnlau96/mvp-eda-microservicios.git
```

7. Branch:

```text
*/main
```

8. Script Path:

```text
Jenkinsfile
```

9. Guardar y ejecutar `Build Now`.

## Webhook de GitHub

Si Jenkins esta expuesto publicamente, configurar webhook:

```text
Payload URL: http://TU_HOST:8081/github-webhook/
Content type: application/json
Events: Just the push event
```

En una laptop local normalmente GitHub no puede alcanzar `localhost`. Para demo local, ejecuta el job manualmente con `Build Now`.

## Que hace el pipeline

1. Checkout del repositorio.
2. Validacion de estructura con `docker compose config --quiet`.
3. Build de imagenes de microservicios.
4. Deploy local con `docker compose up -d`.
5. Smoke test HTTP contra Asistencia, Pagos y Panel.
