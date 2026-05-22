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

Calificar charla:

```cmd
curl -X POST http://localhost:8003/calificar -H "Content-Type: application/json" -d "{\"carnet\":\"201544138\",\"talk_code\":\"SIS-EDA-001\",\"reviewer_type\":\"ESTUDIANTE\",\"rating\":5,\"comment\":\"Excelente charla\"}"
```

Calificacion del profesor al estudiante:

```cmd
curl -X POST http://localhost:8003/calificar -H "Content-Type: application/json" -d "{\"carnet\":\"201544138\",\"talk_code\":\"SIS-EDA-001\",\"reviewer_type\":\"PROFESOR\",\"rating\":5,\"comment\":\"Participacion sobresaliente\"}"
```

Ver calificaciones:

```cmd
curl http://localhost:8003/calificaciones
```

Nota: MySQL queda expuesto en `localhost:3307` para evitar conflictos con instalaciones locales en `3306`.
