import json
import os
import threading
import time
import urllib.request
import urllib.error
import urllib.parse

import pika
import psycopg2
from fastapi import FastAPI, HTTPException
from fastapi.responses import HTMLResponse
from pydantic import BaseModel

DATABASE_URL = os.getenv("DATABASE_URL", "postgresql://asistencia:asistencia@localhost:5432/asistencia_db")
RABBITMQ_URL = os.getenv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
PAGOS_URL = os.getenv("PAGOS_URL", "http://localhost:8002")
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
    input, select, textarea { box-sizing: border-box; width: 100%; padding: 11px; border: 1px solid #c6ccda; border-radius: 6px; }
    textarea { min-height: 74px; resize: vertical; }
    button { margin-top: 16px; width: 100%; padding: 12px; border: 0; border-radius: 6px; background: #0f766e; color: white; font-weight: 700; cursor: pointer; }
    button.secondary { background: #334155; }
    button.danger { background: #b91c1c; }
    .row-actions { display: flex; gap: 8px; }
    .row-actions button { width: auto; margin: 0; padding: 8px 10px; }
    .hidden { display: none; }
    #message, #talkMessage, #studentMessage { margin-top: 14px; padding: 10px; border-radius: 6px; display: none; }
    .ok { display: block !important; background: #dcfce7; color: #166534; }
    .error { display: block !important; background: #fee2e2; color: #991b1b; }
    table { width: 100%; border-collapse: collapse; margin-top: 12px; }
    th, td { padding: 10px; border-bottom: 1px solid #e5e9f2; text-align: left; }
    .pill { display: inline-block; padding: 4px 8px; border-radius: 999px; font-size: 12px; font-weight: 700; }
    .PENDIENTE { background: #fef3c7; color: #92400e; }
    .APROBADO { background: #dcfce7; color: #166534; }
    .CANCELADO { background: #fee2e2; color: #991b1b; }
    .links { margin-top: 12px; display: flex; gap: 10px; flex-wrap: wrap; }
    .links a { color: white; text-decoration: none; border: 1px solid rgba(255,255,255,.5); padding: 8px 10px; border-radius: 6px; }
    .muted { color: #64748b; font-size: 13px; }
    .result-list { display: grid; gap: 8px; margin-top: 10px; }
    .result-list button { margin: 0; text-align: left; background: #ecfdf5; color: #14532d; border: 1px solid #bbf7d0; }
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
      <button type="button" class="secondary" onclick="toggleTalkForm()">Crear / editar charla</button>
      <div id="talkPanel" class="hidden">
      <h2 style="margin-top:16px">Crear charla</h2>
      <form id="talkForm">
        <label>Codigo de charla</label>
        <input id="new_talk_code" value="SIS-EDA-001" />
        <label>Titulo</label>
        <input id="new_talk_title" value="Arquitecturas Dirigidas por Eventos" />
        <label>Carrera</label>
        <select id="new_talk_career">
          <option>Ciencias y Sistemas</option>
          <option>Ingenieria Civil</option>
          <option>Ingenieria Industrial</option>
        </select>
        <button>Crear / actualizar charla</button>
      </form>
      <div id="talkMessage"></div>
      </div>

      <h2>Nueva asistencia</h2>
      <form id="form">
        <label>Buscar estudiante pagado</label>
        <input id="student_search" placeholder="Carnet o nombre" value="201544138" />
        <button type="button" class="secondary" onclick="searchPaidStudents()">Buscar en Pagos</button>
        <div id="studentMessage"></div>
        <div id="studentResults" class="result-list"></div>
        <label>Carnet</label>
        <input id="student_id" value="201544138" readonly />
        <label>Nombre</label>
        <input id="student_name" value="Estudiante Ingenieria" readonly />
        <label>Carrera</label>
        <select id="career" disabled>
          <option>Ciencias y Sistemas</option>
          <option>Ingenieria Civil</option>
          <option>Ingenieria Industrial</option>
        </select>
        <label>Charla</label>
        <select id="talk_id"></select>
        <button>Registrar y publicar evento</button>
      </form>
      <div id="message"></div>
      <button type="button" onclick="fillPending()">Usar estudiante sin pago</button>
    </section>
    <section>
      <h2>Asistencias</h2>
      <h2>Charlas creadas</h2>
      <table>
        <thead><tr><th>Codigo</th><th>Titulo</th><th>Carrera</th><th>Acciones</th></tr></thead>
        <tbody id="talkRows"></tbody>
      </table>
      <h2 style="margin-top:24px">Asistencias</h2>
      <table>
        <thead><tr><th>ID</th><th>Carnet</th><th>Carrera</th><th>Charla</th><th>Estado</th><th>Acciones</th></tr></thead>
        <tbody id="rows"></tbody>
      </table>
    </section>
  </main>
  <script>
    function fillPending() {
      student_search.value = "201544139";
      student_id.value = "";
      student_name.value = "";
      career.value = "Ingenieria Civil";
      talk_id.value = "CIV-EST-001";
      studentMessage.className = "error";
      studentMessage.textContent = "Este carnet esta sembrado como NO PAGADO; al buscarlo no debe aparecer.";
      studentResults.innerHTML = "";
    }
    function escapeHtml(value) {
      return String(value || "").replace(/[&<>"']/g, (char) => ({
        "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;"
      }[char]));
    }
    async function searchPaidStudents() {
      const q = student_search.value.trim();
      if (q.length < 2) {
        studentMessage.className = "error";
        studentMessage.textContent = "Escribe al menos 2 caracteres para buscar.";
        studentResults.innerHTML = "";
        return;
      }
      const response = await fetch("/estudiantes-pagados?q=" + encodeURIComponent(q));
      const data = await response.json();
      if (!response.ok) {
        studentMessage.className = "error";
        studentMessage.textContent = data.detail || "No se pudo consultar Pagos.";
        studentResults.innerHTML = "";
        return;
      }
      if (data.length === 0) {
        studentMessage.className = "error";
        studentMessage.textContent = "No se encontro un estudiante pagado con ese carnet o nombre.";
        studentResults.innerHTML = "";
        return;
      }
      studentMessage.className = "ok";
      studentMessage.textContent = "Selecciona el estudiante para completar el registro.";
      studentResults.innerHTML = data.map(x => `
        <button type="button"
          data-carnet="${escapeHtml(x.carnet)}"
          data-name="${escapeHtml(x.student_name)}"
          data-career="${escapeHtml(x.career)}">
          ${escapeHtml(x.carnet)} - ${escapeHtml(x.student_name)}
          <br><span class="muted">${escapeHtml(x.career)}</span>
        </button>`).join("");
    }
    studentResults.addEventListener("click", (event) => {
      const button = event.target.closest("button[data-carnet]");
      if (!button) return;
      student_id.value = button.dataset.carnet;
      student_name.value = button.dataset.name;
      career.value = button.dataset.career || "Ciencias y Sistemas";
      student_search.value = button.dataset.carnet;
      studentResults.innerHTML = "";
      studentMessage.className = "ok";
      studentMessage.textContent = "Datos cargados desde Pagos. Ya puedes registrar asistencia.";
    });
    function toggleTalkForm() {
      talkPanel.classList.toggle("hidden");
    }
    async function loadTalks() {
      const data = await fetch("/charlas").then(r => r.json());
      const options = data.map(x => `<option value="${x.talk_code}">${x.talk_code} - ${x.talk_title}</option>`).join("");
      talk_id.innerHTML = options;
      talkRows.innerHTML = data.map(x => `
        <tr>
          <td>${x.talk_code}</td>
          <td>${x.talk_title}</td>
          <td>${x.career}</td>
          <td><div class="row-actions">
            <button type="button" onclick='editTalk(${JSON.stringify(x)})'>Editar</button>
            <button type="button" class="danger" onclick='deleteTalk("${x.talk_code}")'>Eliminar</button>
          </div></td>
        </tr>`).join("");
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
          <td><div class="row-actions">
            <button type="button" class="danger" onclick="deleteAttendance(${x.id})">Eliminar</button>
          </div></td>
        </tr>`).join("");
    }
    function editTalk(row) {
      talkPanel.classList.remove("hidden");
      new_talk_code.value = row.talk_code;
      new_talk_title.value = row.talk_title;
      new_talk_career.value = row.career;
    }
    async function deleteTalk(code) {
      if (!confirm("Eliminar charla " + code + "?")) return;
      const response = await fetch("/charlas/" + encodeURIComponent(code), { method: "DELETE" });
      const result = await response.json();
      talkMessage.className = response.ok ? "ok" : "error";
      talkMessage.textContent = response.ok ? "Charla eliminada" : (result.detail || "No se pudo eliminar la charla");
      await loadTalks();
    }
    async function deleteAttendance(id) {
      if (!confirm("Eliminar asistencia #" + id + "?")) return;
      await fetch("/asistencias/" + id, { method: "DELETE" });
      loadRows();
    }
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      const response = await fetch("/asistencia", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          student_id: student_id.value,
          carnet: student_id.value,
          talk_id: talk_id.value,
          talk_code: talk_id.value
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
    talkForm.addEventListener("submit", async (event) => {
      event.preventDefault();
      const response = await fetch("/charlas", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          talk_code: new_talk_code.value,
          talk_title: new_talk_title.value,
          career: new_talk_career.value
        })
      });
      const result = await response.json();
      talkMessage.className = response.ok ? "ok" : "error";
      talkMessage.textContent = response.ok ? "Charla guardada" : (result.detail || "No se pudo guardar la charla");
      await loadTalks();
    });
    loadRows();
    loadTalks();
    searchPaidStudents();
    setInterval(loadRows, 2500);
  </script>
</body>
</html>
"""


class AttendanceIn(BaseModel):
    student_id: str | None = None
    carnet: str | None = None
    student_name: str = ""
    career: str = "Ciencias y Sistemas"
    talk_id: str | None = None
    talk_code: str | None = None
    talk_title: str = "Arquitecturas Dirigidas por Eventos"


class TalkIn(BaseModel):
    talk_code: str
    talk_title: str
    career: str


class TalkRatingIn(BaseModel):
    carnet: str
    talk_code: str
    rating: int
    comment: str = ""


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
                CREATE TABLE IF NOT EXISTS talks (
                  talk_code VARCHAR(50) PRIMARY KEY,
                  talk_title VARCHAR(160) NOT NULL,
                  career VARCHAR(80) NOT NULL,
                  created_at TIMESTAMP NOT NULL DEFAULT NOW()
                )
                """
            )
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
            cur.execute(
                """
                INSERT INTO talks (talk_code, talk_title, career) VALUES
                ('SIS-EDA-001', 'Arquitecturas Dirigidas por Eventos', 'Ciencias y Sistemas'),
                ('CIV-EST-001', 'Estructuras y gestion de obra', 'Ingenieria Civil'),
                ('IND-PRO-001', 'Optimizacion de procesos industriales', 'Ingenieria Industrial')
                ON CONFLICT (talk_code) DO NOTHING
                """
            )
            cur.execute(
                """
                CREATE TABLE IF NOT EXISTS talk_ratings (
                  id SERIAL PRIMARY KEY,
                  student_id VARCHAR(50) NOT NULL,
                  talk_id VARCHAR(50) NOT NULL,
                  rating INT NOT NULL CHECK (rating BETWEEN 1 AND 5),
                  comment VARCHAR(255),
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
                cur.execute(
                    """
                    CREATE UNIQUE INDEX IF NOT EXISTS ux_talk_ratings_student_talk
                    ON talk_ratings (student_id, talk_id)
                    """
                )
            except Exception as exc:
                conn.rollback()
                print(f"No se pudo crear indice unico por datos existentes duplicados: {exc}")

    with get_db() as conn:
        with conn.cursor() as cur:
            try:
                cur.execute(
                    """
                    CREATE UNIQUE INDEX IF NOT EXISTS ux_talk_ratings_student_talk
                    ON talk_ratings (student_id, talk_id)
                    """
                )
            except Exception as exc:
                conn.rollback()
                print(f"No se pudo crear indice unico de calificaciones: {exc}")


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


def verify_payment(carnet):
    try:
        with urllib.request.urlopen(f"{PAGOS_URL}/verificar-pago/{carnet}", timeout=5) as response:
            payload = json.loads(response.read().decode("utf-8"))
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError) as exc:
        raise HTTPException(status_code=503, detail=f"No se pudo verificar el pago en este momento: {exc}")

    if not payload.get("paid"):
        reason = payload.get("reason", "El estudiante no aparece como pagado")
        raise HTTPException(status_code=402, detail=f"No se puede registrar asistencia: {reason}")

    return payload


def search_paid_students(q):
    encoded = urllib.parse.quote(q)
    try:
        with urllib.request.urlopen(f"{PAGOS_URL}/estudiantes-pagados?q={encoded}", timeout=5) as response:
            return json.loads(response.read().decode("utf-8"))
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError) as exc:
        raise HTTPException(status_code=503, detail=f"No se pudo consultar estudiantes pagados: {exc}")


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

    payment = verify_payment(carnet)
    paid_name = payment.get("student_name") or data.student_name
    paid_career = payment.get("career") or data.career

    with get_db() as conn:
        with conn.cursor() as cur:
            cur.execute(
                "SELECT talk_title, career FROM talks WHERE talk_code = %s",
                (talk_code,),
            )
            talk = cur.fetchone()
            if not talk:
                raise HTTPException(status_code=400, detail=f"La charla {talk_code} no existe. Debe crearse antes de registrar asistencia.")
            talk_title, talk_career = talk
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
                    (carnet, paid_name, paid_career, talk_code, talk_title),
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
        "student_name": paid_name,
        "career": paid_career,
        "talk_id": talk_code,
        "talk_code": talk_code,
        "talk_title": talk_title,
        "talk_career": talk_career,
    }
    publish_event("AsistenciaRegistrada", event)
    return {"attendance_id": attendance_id, "status": status, "event": "AsistenciaRegistrada"}


@app.get("/estudiantes-pagados")
def paid_students(q: str = ""):
    if len(q.strip()) < 2:
        return []
    return search_paid_students(q.strip())


@app.post("/calificar-charla")
def rate_talk(data: TalkRatingIn):
    if data.rating < 1 or data.rating > 5:
        raise HTTPException(status_code=400, detail="rating debe estar entre 1 y 5")

    with get_db() as conn:
        with conn.cursor() as cur:
            cur.execute(
                """
                SELECT id FROM attendance
                WHERE student_id = %s AND talk_id = %s AND status = 'APROBADO'
                LIMIT 1
                """,
                (data.carnet, data.talk_code),
            )
            if not cur.fetchone():
                raise HTTPException(
                    status_code=400,
                    detail="Solo se puede calificar una charla si el carnet tiene asistencia APROBADA",
                )
            cur.execute(
                "SELECT id FROM talk_ratings WHERE student_id = %s AND talk_id = %s LIMIT 1",
                (data.carnet, data.talk_code),
            )
            existing = cur.fetchone()
            if existing:
                rating_id = existing[0]
                cur.execute(
                    """
                    UPDATE talk_ratings
                    SET rating = %s, comment = %s, created_at = NOW()
                    WHERE id = %s
                    """,
                    (data.rating, data.comment, rating_id),
                )
            else:
                cur.execute(
                    """
                    INSERT INTO talk_ratings (student_id, talk_id, rating, comment)
                    VALUES (%s, %s, %s, %s)
                    RETURNING id
                    """,
                    (data.carnet, data.talk_code, data.rating, data.comment),
                )
                rating_id = cur.fetchone()[0]

    return {"ok": True, "rating_id": rating_id}


@app.get("/", response_class=HTMLResponse)
def home():
    return HTML


@app.post("/charlas")
def upsert_talk(data: TalkIn):
    allowed = {"Ingenieria Civil", "Ingenieria Industrial", "Ciencias y Sistemas"}
    if data.career not in allowed:
        raise HTTPException(status_code=400, detail="Carrera invalida")
    if not data.talk_code.strip() or not data.talk_title.strip():
        raise HTTPException(status_code=400, detail="Codigo y titulo son obligatorios")

    with get_db() as conn:
        with conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO talks (talk_code, talk_title, career)
                VALUES (%s, %s, %s)
                ON CONFLICT (talk_code)
                DO UPDATE SET talk_title = EXCLUDED.talk_title, career = EXCLUDED.career
                """,
                (data.talk_code.strip(), data.talk_title.strip(), data.career),
            )
    return {"ok": True}


@app.get("/charlas")
def list_talks():
    with get_db() as conn:
        with conn.cursor() as cur:
            cur.execute("SELECT talk_code, talk_title, career, created_at FROM talks ORDER BY talk_code")
            rows = cur.fetchall()
    return [
        {
            "talk_code": row[0],
            "talk_id": row[0],
            "talk_title": row[1],
            "career": row[2],
            "created_at": row[3].isoformat(),
        }
        for row in rows
    ]


@app.delete("/charlas/{talk_code}")
def delete_talk(talk_code: str):
    with get_db() as conn:
        with conn.cursor() as cur:
            cur.execute("SELECT COUNT(*) FROM attendance WHERE talk_id = %s", (talk_code,))
            if cur.fetchone()[0] > 0:
                raise HTTPException(status_code=409, detail="No se puede eliminar una charla que ya tiene asistencias")
            cur.execute("DELETE FROM talks WHERE talk_code = %s", (talk_code,))
            if cur.rowcount == 0:
                raise HTTPException(status_code=404, detail="Charla no encontrada")
    return {"ok": True}


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


@app.delete("/asistencias/{attendance_id}")
def delete_attendance(attendance_id: int):
    with get_db() as conn:
        with conn.cursor() as cur:
            cur.execute("DELETE FROM attendance WHERE id = %s", (attendance_id,))
            if cur.rowcount == 0:
                raise HTTPException(status_code=404, detail="Asistencia no encontrada")
    return {"ok": True}


@app.get("/calificaciones-charla")
def list_talk_ratings():
    with get_db() as conn:
        with conn.cursor() as cur:
            cur.execute(
                """
                SELECT student_id, talk_id, rating, comment, created_at
                FROM talk_ratings
                ORDER BY created_at DESC
                LIMIT 100
                """
            )
            rows = cur.fetchall()
    return [
        {
            "carnet": row[0],
            "student_id": row[0],
            "talk_code": row[1],
            "talk_id": row[1],
            "rating": row[2],
            "comment": row[3],
            "created_at": row[4].isoformat(),
        }
        for row in rows
    ]
