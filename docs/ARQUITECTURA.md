# Documentacion tecnica del MVP

## Objetivo

Demostrar una arquitectura de microservicios poliglota con comunicacion basada en eventos, consistencia eventual, bases de datos independientes, Docker Compose, Jenkins y despliegue automatizable con Ansible.

## Microservicios

| Servicio | Tecnologia | Base de datos | Puerto | Responsabilidad |
| --- | --- | --- | --- | --- |
| Asistencia | Python + FastAPI | PostgreSQL | 8001 | Registra asistencia por carnet y charla. |
| Pagos | Node.js + Express | MongoDB | 8002 | Verifica si el carnet tiene pago activo. |
| Panel | Go | MySQL | 8003 | Replica aprobados y permite consulta filtrada para profesores. |

Servicio de soporte:

| Servicio | Tecnologia | Puerto | Responsabilidad |
| --- | --- | --- | --- |
| Keycloak | Identity Provider | 8080 | Usuarios, roles y emision de tokens JWT. |
| Jenkins | CI/CD | 8081 | Ejecuta pipeline de build, test, deploy y smoke test. |

## Bus de eventos

RabbitMQ usa un exchange tipo `topic` llamado `mvp_eventos`.

Eventos principales:

| Evento | Publicador | Consumidores | Proposito |
| --- | --- | --- | --- |
| `AsistenciaRegistrada` | Asistencia | Pagos | Solicitar verificacion de pago. |
| `PagoVerificado` | Pagos | Asistencia, Panel | Aprobar asistencia y replicar al panel. |
| `PagoRechazado` | Pagos | Asistencia | Cancelar asistencia pendiente. |

## Flujo funcional

1. Asistencia administra el catalogo de charlas en la tabla `talks`.
2. El estudiante registra asistencia seleccionando una charla existente.
3. Asistencia valida el pago de forma sincrona contra Pagos.
4. Si el carnet no esta pagado, Asistencia rechaza el registro con `402`.
5. Asistencia valida que no exista otro registro para el mismo `carnet + talk_code`.
6. Asistencia guarda estado `PENDIENTE` en PostgreSQL.
7. Asistencia publica `AsistenciaRegistrada`.
8. Pagos consulta MongoDB y publica `PagoVerificado`.
9. Asistencia actualiza el estado a `APROBADO`.
10. Panel escucha `PagoVerificado` y replica el registro aprobado en MySQL.
11. El Panel permite a profesores consultar asistentes por charla, carrera, carnet o nombre.
12. El Panel carga el catalogo de charlas desde Asistencia y permite al profesor calificar charlas.

## Persistencia y consistencia

Cada microservicio tiene su propia base de datos:

- PostgreSQL: tabla `attendance`.
- PostgreSQL: tabla `talks`.
- MongoDB: coleccion `payments`.
- MySQL: tablas `approved_attendance`, `ratings` y `professor_talk_ratings`.

La consistencia entre servicios es eventual. El estado inicial queda `PENDIENTE` y luego cambia cuando llegan los eventos de respuesta desde Pagos.

Para evitar duplicados, Asistencia aplica:

- Validacion en aplicacion antes de insertar.
- Indice unico `ux_attendance_student_talk` en PostgreSQL para instalaciones limpias.

## Tabla espejo

`approved_attendance` en MySQL funciona como tabla espejo para el Panel. No es la fuente principal de asistencia; es una proyeccion consumidora del evento `PagoVerificado`.

## Seguridad aplicada en el MVP

Este es un prototipo local, no una configuracion productiva. Aun asi, se aplican buenas practicas basicas:

- Keycloak como proveedor de identidad con realm `mvp-eda`.
- Roles de demo: `student`, `teacher`, `admin`.
- Cliente OIDC `mvp-eda-api` para obtener tokens JWT.
- Credenciales por variables de entorno en `docker-compose.yml`.
- Servicios aislados por red interna de Docker Compose.
- Endpoints de escritura con validaciones de dominio.
- Validacion de duplicados para integridad de datos.
- Login administrativo en Pagos para evitar que estudiantes modifiquen pagos.
- Verificacion previa de pago antes de registrar asistencia.
- Filtros y limites de consulta en Panel para manejar grupos grandes.
- Separacion de bases de datos por microservicio.

Mejoras recomendadas para produccion:

- HTTPS con proxy inverso.
- Validacion obligatoria de JWT en cada endpoint protegido.
- API Gateway para centralizar autenticacion, CORS y rate limiting.
- Roles para estudiantes, profesores y administradores.
- Secretos en Jenkins Credentials, Vault o Docker Secrets.
- CORS restrictivo por dominio.
- Usuarios RabbitMQ y bases de datos no compartidos.

## CI/CD

El `Jenkinsfile` incluye tres etapas:

1. Checkout.
2. Validate de `docker compose`.
3. Lint/Test ligero de estructura.
4. Build de imagenes Docker.
5. Deploy con Docker Compose.
6. Smoke test HTTP.

Para una entrega mas completa se puede conectar Jenkins con GitHub Webhooks y agregar pruebas por servicio.

## Dockerizacion y orquestacion

`docker-compose.yml` levanta:

- RabbitMQ con consola de administracion.
- PostgreSQL.
- MongoDB.
- MySQL.
- Los tres microservicios.

Puertos principales:

```text
8001  Asistencia
8002  Pagos
8003  Panel
15672 RabbitMQ Management
8080  Keycloak
8081  Jenkins
3307  MySQL local
```

## Despliegue con Ansible

La carpeta `ansible/` incluye:

- `inventory.ini`: servidor destino.
- `deploy.yml`: instala Docker, copia el proyecto y ejecuta Docker Compose.
- `stop.yml`: detiene el stack.
- `group_vars/demo.yml`: variables de puertos y ruta destino.

## Endpoints principales

```text
POST /asistencia
GET  /asistencias
DELETE /asistencias/{id}
GET  /charlas
POST /charlas
DELETE /charlas/{talk_code}

GET  /pagos
POST /pagos
DELETE /pagos/:student_id
GET  /verificar-pago/:student_id

GET  /panel
DELETE /panel/{attendance_id}
GET  /charlas
POST /calificar-charla
GET  /calificaciones-charla
DELETE /calificaciones-charla/{id}
```

## Caso de prueba principal

Carnet con pago:

```text
201544138
Carrera: Ciencias y Sistemas
Charla: SIS-EDA-001
```

Resultado esperado:

```text
PENDIENTE -> PagoVerificado -> APROBADO -> visible en Panel
```

Si se intenta registrar otra vez el mismo carnet para `SIS-EDA-001`, el servicio responde `409 Conflict`.
