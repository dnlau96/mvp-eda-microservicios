# Seguridad con Keycloak

Este MVP incluye Keycloak como proveedor de identidad para demostrar autenticacion, tokens JWT y roles.

## Levantar

```cmd
docker compose up -d keycloak
```

Consola:

```text
http://localhost:8080
usuario: admin
password: admin
```

Realm importado:

```text
mvp-eda
```

Cliente:

```text
mvp-eda-api
```

## Usuarios de demo

| Usuario | Password | Rol |
| --- | --- | --- |
| `201544138` | `estudiante123` | `student` |
| `profesor` | `profesor123` | `teacher` |
| `admin-demo` | `admin123` | `admin` |

## Obtener token JWT

En Windows `cmd.exe`:

```cmd
curl -X POST http://localhost:8080/realms/mvp-eda/protocol/openid-connect/token -H "Content-Type: application/x-www-form-urlencoded" -d "client_id=mvp-eda-api" -d "grant_type=password" -d "username=201544138" -d "password=estudiante123"
```

Para profesor:

```cmd
curl -X POST http://localhost:8080/realms/mvp-eda/protocol/openid-connect/token -H "Content-Type: application/x-www-form-urlencoded" -d "client_id=mvp-eda-api" -d "grant_type=password" -d "username=profesor" -d "password=profesor123"
```

La respuesta incluye:

```json
{
  "access_token": "...",
  "expires_in": 300,
  "token_type": "Bearer"
}
```

## Como explicarlo en la demo

- Keycloak centraliza usuarios, credenciales y roles.
- Los microservicios podrian validar el `access_token` en cada endpoint protegido.
- `student` registra asistencia y califica charlas.
- `teacher` califica estudiantes.
- `admin` administra catalogos, charlas y usuarios.

En este MVP se incluye Keycloak como pieza visible de seguridad. La validacion estricta de tokens en cada microservicio se puede activar como siguiente mejora sin cambiar la arquitectura.
