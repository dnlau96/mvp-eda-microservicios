# Deploy con Ansible

Este playbook instala Docker en una maquina Ubuntu/Debian, copia el proyecto y levanta el MVP con Docker Compose.

## 1. Requisitos

En tu maquina local necesitas Ansible. En Windows, lo mas practico es usar WSL con Ubuntu.

```bash
sudo apt update
sudo apt install -y ansible rsync
```

La maquina destino debe permitir acceso SSH y, si usa Ubuntu/Debian, tener Python 3 disponible.

## 2. Configurar inventario

Edita `inventory.ini`:

```ini
[demo]
192.168.1.50 ansible_user=ubuntu
```

Si usas llave SSH:

```ini
[demo]
192.168.1.50 ansible_user=ubuntu ansible_ssh_private_key_file=~/.ssh/id_rsa
```

## 3. Ejecutar deploy

Desde la raiz del proyecto:

```bash
ansible-playbook -i ansible/inventory.ini ansible/deploy.yml
```

Validar sintaxis antes del deploy:

```bash
ansible-playbook --syntax-check -i ansible/inventory.ini ansible/deploy.yml
```

## 4. Abrir la demo

Reemplaza la IP por la de tu servidor:

```text
http://192.168.1.50:8001  Asistencia
http://192.168.1.50:8002  Pagos
http://192.168.1.50:8003  Panel profesores
http://192.168.1.50:15672 RabbitMQ
```

RabbitMQ usa:

```text
usuario: guest
password: guest
```

## 5. Detener

```bash
ansible-playbook -i ansible/inventory.ini ansible/stop.yml
```

## Notas

- El puerto publico de MySQL se deja en `3307` para evitar conflicto con MySQL local.
- Puedes cambiar puertos en `group_vars/demo.yml`.
- Este playbook esta hecho para Ubuntu/Debian. Para Windows Server o Red Hat habria que adaptar la instalacion de Docker.
