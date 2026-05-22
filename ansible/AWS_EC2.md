# Deploy en Amazon EC2

Guia rapida para correr el MVP en una instancia de prueba de AWS usando Docker Compose y Ansible.

## 1. Crear la instancia EC2

Recomendacion para demo:

- AMI: Ubuntu Server 24.04 LTS o 22.04 LTS.
- Tipo: `t3.medium` o superior. El proyecto levanta varias bases de datos, Jenkins y Keycloak; `t2.micro` suele quedarse corto.
- Disco: 20 GB gp3.
- Key pair: crea o usa una llave `.pem`.

## 2. Security Group

Abre solo lo necesario para la demo:

```text
22     SSH                 Tu IP publica
8001   Asistencia          Tu IP publica o 0.0.0.0/0 para demo temporal
8002   Pagos               Tu IP publica o 0.0.0.0/0 para demo temporal
8003   Panel profesores    Tu IP publica o 0.0.0.0/0 para demo temporal
15672  RabbitMQ UI         Tu IP publica
8081   Jenkins             Tu IP publica
8080   Keycloak            Tu IP publica
```

Evita abrir puertos de bases de datos a internet (`5432`, `27017`, `3307`) salvo que tengas una razon concreta.

## 3. Preparar Ansible desde Windows

Lo mas simple en Windows es usar WSL Ubuntu.

```bash
sudo apt update
sudo apt install -y ansible rsync
```

Copia tu llave `.pem` a WSL, por ejemplo:

```bash
mkdir -p ~/.ssh
cp /mnt/c/Users/TU_USUARIO/Downloads/mvp-eda-demo.pem ~/.ssh/
chmod 400 ~/.ssh/mvp-eda-demo.pem
```

## 4. Configurar inventario

Copia el ejemplo:

```bash
cp ansible/inventory.aws.example.ini ansible/inventory.aws.ini
```

Edita `ansible/inventory.aws.ini`:

```ini
[demo]
18.222.111.222 ansible_user=ubuntu ansible_ssh_private_key_file=~/.ssh/mvp-eda-demo.pem

[demo:vars]
ansible_python_interpreter=/usr/bin/python3
```

Prueba conexion:

```bash
ansible -i ansible/inventory.aws.ini demo -m ping
```

## 5. Desplegar

Desde la raiz del proyecto:

```bash
ansible-playbook -i ansible/inventory.aws.ini ansible/deploy.yml
```

Al finalizar, abre:

```text
http://EC2_PUBLIC_IP:8001   Asistencia
http://EC2_PUBLIC_IP:8002   Pagos
http://EC2_PUBLIC_IP:8003   Panel profesores
http://EC2_PUBLIC_IP:15672  RabbitMQ
http://EC2_PUBLIC_IP:8080   Keycloak
http://EC2_PUBLIC_IP:8081   Jenkins
```

Credenciales demo:

```text
Pagos:      admin / admin123
Profesores: profesor / profesor123
RabbitMQ:   guest / guest
Keycloak:   admin / admin
```

## 6. Comandos utiles

Ver contenedores:

```bash
ssh -i ~/.ssh/mvp-eda-demo.pem ubuntu@EC2_PUBLIC_IP
cd /opt/mvp-eda
sudo docker compose -p mvp_eda ps
```

Ver logs:

```bash
sudo docker compose -p mvp_eda logs -f asistencia-service
sudo docker compose -p mvp_eda logs -f pagos-service
sudo docker compose -p mvp_eda logs -f panel-service
```

Detener la demo:

```bash
ansible-playbook -i ansible/inventory.aws.ini ansible/stop.yml
```

Para evitar costos, detiene o termina la instancia cuando ya no la uses.
