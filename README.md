# MVP EDA Microservicios

Prototipo minimo con 3 microservicios poliglotas, RabbitMQ, PostgreSQL, MongoDB, MySQL y Jenkinsfile.
Incluye Keycloak para demostrar autenticacion, JWT y roles.

## Levantar

```bash
docker compose up --build
```

RabbitMQ UI: http://localhost:15672 (`guest` / `guest`)
Keycloak UI: http://localhost:8080 (`admin` / `admin`)

Documentacion tecnica de arquitectura: [docs/ARQUITECTURA.md](docs/ARQUITECTURA.md)
Guia de seguridad Keycloak: [security/README.md](security/README.md)
Guia CI/CD con Jenkins: [jenkins/README.md](jenkins/README.md)

## Jenkins CI/CD

Levantar Jenkins:

```cmd
docker compose up -d --build jenkins
```

Abrir:

```text
http://localhost:8081
```

Password inicial:

```cmd
docker compose exec jenkins cat /var/jenkins_home/secrets/initialAdminPassword
```

Crear un Pipeline usando este repositorio:

```text
https://github.com/dnlau96/mvp-eda-microservicios.git
```

Script Path:

```text
Jenkinsfile
```

## Probar flujo

En Windows `cmd.exe`, usa los comandos en una sola linea:

El carnet principal de prueba es `201544138` y pertenece a `Ciencias y Sistemas`.

Carreras disponibles:

```text
Ingenieria Civil
Ingenieria Industrial
Ciencias y Sistemas
```

Charlas sugeridas:

```text
SIS-EDA-001  Arquitecturas Dirigidas por Eventos
CIV-EST-001  Estructuras y gestion de obra
IND-PRO-001  Optimizacion de procesos industriales
```

Carnet con pago aprobado:

```cmd
curl -X POST http://localhost:8001/asistencia -H "Content-Type: application/json" -d "{\"carnet\":\"201544138\",\"student_name\":\"Estudiante Ingenieria\",\"career\":\"Ciencias y Sistemas\",\"talk_code\":\"SIS-EDA-001\",\"talk_title\":\"Arquitecturas Dirigidas por Eventos\"}"
```

Si ejecutas el mismo comando dos veces, la segunda respuesta debe ser `409 Conflict` porque no se permite duplicar asistencia para el mismo carnet y charla.

Carnet sin pago:

```cmd
curl -X POST http://localhost:8001/asistencia -H "Content-Type: application/json" -d "{\"carnet\":\"201544139\",\"student_name\":\"Estudiante Pendiente\",\"career\":\"Ingenieria Civil\",\"talk_code\":\"CIV-EST-001\",\"talk_title\":\"Estructuras y gestion de obra\"}"
```

Ver asistencias:

```cmd
curl http://localhost:8001/asistencias
```

Ver panel de aprobados:

```cmd
curl http://localhost:8003/panel
```

Calificar charla desde el microservicio de Asistencia:

```cmd
curl -X POST http://localhost:8001/calificar-charla -H "Content-Type: application/json" -d "{\"carnet\":\"201544138\",\"talk_code\":\"SIS-EDA-001\",\"rating\":5,\"comment\":\"Excelente charla\"}"
```

Ver calificaciones de charlas:

```cmd
curl http://localhost:8001/calificaciones-charla
```

Filtrar panel de profesores por charla:

```cmd
curl "http://localhost:8003/panel?talk_code=SIS-EDA-001&limit=50"
```

Nota: MySQL queda expuesto en `localhost:3307` para evitar conflictos con instalaciones locales en `3306`.

## Pagos administrativos

El microservicio de Pagos esta protegido con login para que un estudiante no pueda marcarse como pagado.

```text
http://localhost:8002
usuario: admin
password: admin123
```

Desde esa pantalla se puede crear, editar o eliminar pagos.
