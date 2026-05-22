import json
import os
import threading
import time

import pika
import psycopg2
from fastapi import FastAPI, HTTPException
from fastapi.responses import HTMLResponse
from pydantic import BaseModel

DATABASE_URL = os.getenv("DATABASE_URL", "postgresql://asistencia:asistencia@localhost:5432/asistencia_db")
RABBITMQ_URL = os.getenv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
EXCHANGE = "mvp_eventos"

app = FastAPI(title="Asistencia Service")

HTML = """
<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Asistencia | MVP EDA</title>
  <style>
    body { margin: 0; font-family: Arial, sans-serif; background: #f6f7fb; color: #172033; }
    header { background: #0f766e; color: white; padding: 28px 36px; }
    main { max-width: 1080px; margin: 28px auto; padding: 0 20px; }
    .grid { display: grid; grid-template-columns: 360px 1fr; gap: 20px; align-items: start; }
    section { background: white; border: 1px solid #d9dee8; border-radius: 8px; padding: 20px; }
    h1, h2 { margin: 0 0 10px; }
    label { display: block; margin: 14px 0 6px; font-weight: 700; }
    input { box-sizing: border-box; width: 100%; padding: 11px; border: 1px solid #c6ccda; border-radius: 6px; }
    button { margin-top: 16px; width: 100%; padding: 12px; border: 0; border-radius: 6px; background: #0f766e; color: white; font-weight: 700; cursor: pointer; }
    #message { margin-top: 14px; padding: 10px; border-radius: 6px; display: none; }
    #message.ok { display: block; background: #dcfce7; color: #166534; }
    #message.error { display: block; background: #fee2e2; color: #991b1b; }
    table { width: 100%; border-collapse: collapse; margin-top: 12px; }
    th, td { padding: 10px; border-bottom: 1px solid #e5e9f2; text-align: left; }
    .pill { display: inline-block; padding: 4px 8px; border-radius: 999px; font-size: 12px; font-weight: 700; }
    .PENDIENTE { background: #fef3c7; color: #92400e; }
    .APROBADO { background: #dcfce7; color: #166534; }
    .CANCELADO { background: #fee2e2; color: #991b1b; }
    .links { margin-top: 12px; display: flex; gap: 10px; flex-wrap: wrap; }
    .links a { color: white; text-decoration: none; border: 1px solid rgba(255,255,255,.5); padding: 8px 10px; border-radius: 6px; }
    @media (max-width: 820px) { .grid { grid-template-columns: 1fr; } }
  </style>
</head>
<body>
  <header>
    <h1>Microservicio de Asistencia</h1>
    <p>Registra asistencia por carnet, carrera y charla; publica AsistenciaRegistrada y actualiza estado segun eventos de pago.</p>
    <div class="links">
      <a href="http://localhost:8002">Pagos</a>
      <a href="http://localhost:8003">Panel profesores</a>
      <a href="http://localhost:15672">RabbitMQ</a>
    </div>
  </header>
  <main class="grid">
    <section>
      <h2>Nueva asistencia</h2>
      <form id="form">
        <label>Carnet</label>
        <input id="student_id" value="201544138" />
        <label>Nombre</label>
        <input id="student_name" value="Estudiante Ingenieria" />
        <label>Carrera</label>
        <input id="career" list="careers" value="Ciencias y Sistemas" />
        <datalist id="careers">
          <option value="Ingenieria Civil"></option>
          <option value="Ingenieria Industrial"></option>
          <option value="Ciencias y Sistemas"></option>
        </datalist>
        <label>Codigo de charla</label>
        <input id="talk_id" value="SIS-EDA-001" />
        <label>Titulo de charla</label>
        <input id="talk_title" value="Arquitecturas Dirigidas por Eventos" />
        <button>Registrar y publicar evento</button>
      </form>
      <div id="message"></div>
      <button type="button" onclick="fillPending()">Usar estudiante sin pago</button>
    </section>
    <section>
      <h2>Asistencias</h2>
      <table>
        <thead><tr><th>ID</th><th>Carnet</th><th>Carrera</th><th>Charla</th><th>Estado</th></tr></thead>
        <tbody id="rows"></tbody>
      </table>
    </section>
  </main>
  <script>
    function fillPending() {
      student_id.value = "stu-002";
      student_name.value = "Luis Pendiente";
      career.value = "Ingenieria Civil";
      talk_id.value = "CIV-EST-001";
      talk_title.value = "Estructuras y gestion de obra";
    }
    async function loadRows() {
      const data = await fetch("/asistencias").then(r => r.json());
      rows.innerHTML = data.map(x => `
        <tr>
          <td>${x.id}</td>
          <td>${x.student_name}<br><small>${x.carnet}</small></td>
          <td>${x.career}</td>
          <td>${x.talk_code}<br><small>${x.talk_title}</small></td>
          <td><span class="pill ${x.status}">${x.status}</span></td>
        </tr>`).join("");
    }
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      const response = await fetch("/asistencia", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          student_id: student_id.value,
          carnet: student_id.value,
          student_name: student_name.value,
          career: career.value,
          talk_id: talk_id.value,
          talk_code: talk_id.value,
          talk_title: talk_title.value
        })
      });
      const result = await response.json();
      message.className = response.ok ? "ok" : "error";
      message.textContent = response.ok
        ? "Asistencia registrada. Evento publicado: " + result.event
        : (result.detail || result.error || "No se pudo registrar la asistencia");
      setTimeout(loadRows, 1200);
      loadRows();
    });
    loadRows();
    setInterval(loadRows, 2500);
  </script>
</body>
</html>
"""


class AttendanceIn(BaseModel):
    student_id: str | None = None
    carnet: str | None = None
    student_name: str
    career: str = "Ciencias y Sistemas"
    talk_id: str | None = None
    talk_code: str | None = None
    talk_title: str = "Arquitecturas Dirigidas por Eventos"


def wait_for_postgres():
    while True:
        try:
            conn = psycopg2.connect(DATABASE_URL)
            conn.close()
            return
        except Exception as exc:
            print(f"Postgres no listo: {exc}")
            time.sleep(2)


def get_db():
    return psycopg2.connect(DATABASE_URL)


def init_db():
    with get_db() as conn:
        with conn.cursor() as cur:
            cur.execute(
                """
                CREATE TABLE IF NOT EXISTS attendance (
                  id SERIAL PRIMARY KEY,
                  student_id VARCHAR(50) NOT NULL,
                  student_name VARCHAR(120) NOT NULL,
                  career VARCHAR(80) NOT NULL DEFAULT 'Ciencias y Sistemas',
                  talk_id VARCHAR(50) NOT NULL,
                  talk_title VARCHAR(160) NOT NULL DEFAULT 'Arquitecturas Dirigidas por Eventos',
                  status VARCHAR(20) NOT NULL DEFAULT 'PENDIENTE',
                  created_at TIMESTAMP NOT NULL DEFAULT NOW()
                )
                """
            )
            cur.execute("ALTER TABLE attendance ADD COLUMN IF NOT EXISTS career VARCHAR(80) NOT NULL DEFAULT 'Ciencias y Sistemas'")
            cur.execute("ALTER TABLE attendance ADD COLUMN IF NOT EXISTS talk_title VARCHAR(160) NOT NULL DEFAULT 'Arquitecturas Dirigidas por Eventos'")

    with get_db() as conn:
        with conn.cursor() as cur:
            try:
                cur.execute(
                    """
                    CREATE UNIQUE INDEX IF NOT EXISTS ux_attendance_student_talk
                    ON attendance (student_id, talk_id)
                    """
                )
            except Exception as exc:
                conn.rollback()
                print(f"No se pudo crear indice unico por datos existentes duplicados: {exc}")


def rabbit_channel():
    params = pika.URLParameters(RABBITMQ_URL)
    conn = pika.BlockingConnection(params)
    channel = conn.channel()
    channel.exchange_declare(exchange=EXCHANGE, exchange_type="topic", durable=True)
    return conn, channel


def publish_event(event_type, payload):
    conn, channel = rabbit_channel()
    channel.basic_publish(
        exchange=EXCHANGE,
        routing_key=event_type,
        body=json.dumps(payload).encode("utf-8"),
        properties=pika.BasicProperties(content_type="application/json", delivery_mode=2),
    )
    conn.close()


def handle_payment_response(event_type, body):
    payload = json.loads(body)
    status = "APROBADO" if event_type == "PagoVerificado" else "CANCELADO"
    with get_db() as conn:
        with conn.cursor() as cur:
            cur.execute(
                "UPDATE attendance SET status = %s WHERE id = %s",
                (status, payload["attendance_id"]),
            )
    print(f"Asistencia {payload['attendance_id']} actualizada a {status}")


def consume_payment_events():
    while True:
        try:
            conn, channel = rabbit_channel()
            queue = channel.queue_declare(queue="asistencia.payment.responses", durable=True).method.queue
            channel.queue_bind(exchange=EXCHANGE, queue=queue, routing_key="PagoVerificado")
            channel.queue_bind(exchange=EXCHANGE, queue=queue, routing_key="PagoRechazado")

            def callback(ch, method, properties, body):
                handle_payment_response(method.routing_key, body)
                ch.basic_ack(delivery_tag=method.delivery_tag)

            channel.basic_consume(queue=queue, on_message_callback=callback)
            print("Asistencia escuchando PagoVerificado/PagoRechazado")
            channel.start_consuming()
        except Exception as exc:
            print(f"Consumer asistencia reiniciando: {exc}")
            time.sleep(3)


@app.on_event("startup")
def startup():
    wait_for_postgres()
    init_db()
    threading.Thread(target=consume_payment_events, daemon=True).start()


@app.post("/asistencia")
def register_attendance(data: AttendanceIn):
    carnet = data.carnet or data.student_id
    talk_code = data.talk_code or data.talk_id
    if not carnet or not talk_code:
        raise HTTPException(status_code=400, detail="carnet y talk_code son obligatorios")

    with get_db() as conn:
        with conn.cursor() as cur:
            cur.execute(
                """
                SELECT id, status FROM attendance
                WHERE student_id = %s AND talk_id = %s
                LIMIT 1
                """,
                (carnet, talk_code),
            )
            existing = cur.fetchone()
            if existing:
                raise HTTPException(
                    status_code=409,
                    detail=f"El carnet {carnet} ya tiene asistencia registrada para la charla {talk_code} con estado {existing[1]}",
                )
            try:
                cur.execute(
                    """
                    INSERT INTO attendance (student_id, student_name, career, talk_id, talk_title, status)
                    VALUES (%s, %s, %s, %s, %s, 'PENDIENTE')
                    RETURNING id, status
                    """,
                    (carnet, data.student_name, data.career, talk_code, data.talk_title),
                )
            except psycopg2.IntegrityError:
                raise HTTPException(
                    status_code=409,
                    detail=f"El carnet {carnet} ya tiene asistencia registrada para la charla {talk_code}",
                )
            attendance_id, status = cur.fetchone()

    event = {
        "attendance_id": attendance_id,
        "student_id": carnet,
        "carnet": carnet,
        "student_name": data.student_name,
        "career": data.career,
        "talk_id": talk_code,
        "talk_code": talk_code,
        "talk_title": data.talk_title,
    }
    publish_event("AsistenciaRegistrada", event)
    return {"attendance_id": attendance_id, "status": status, "event": "AsistenciaRegistrada"}


@app.get("/", response_class=HTMLResponse)
def home():
    return HTML


@app.get("/asistencias")
def list_attendance():
    with get_db() as conn:
        with conn.cursor() as cur:
            cur.execute(
                "SELECT id, student_id, student_name, career, talk_id, talk_title, status, created_at FROM attendance ORDER BY id"
            )
            rows = cur.fetchall()
    return [
        {
            "id": row[0],
            "student_id": row[1],
            "carnet": row[1],
            "student_name": row[2],
            "career": row[3],
            "talk_id": row[4],
            "talk_code": row[4],
            "talk_title": row[5],
            "status": row[6],
            "created_at": row[7].isoformat(),
        }
        for row in rows
    ]
