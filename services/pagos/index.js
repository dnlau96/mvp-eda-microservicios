const amqp = require("amqplib");
const express = require("express");
const { MongoClient } = require("mongodb");

const RABBITMQ_URL = process.env.RABBITMQ_URL || "amqp://guest:guest@localhost:5672/";
const MONGO_URL = process.env.MONGO_URL || "mongodb://localhost:27017/pagos_db";
const EXCHANGE = "mvp_eventos";

const app = express();
app.use(express.json());

let payments;
let channel;

const html = `<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Pagos | MVP EDA</title>
  <style>
    body { margin: 0; font-family: Arial, sans-serif; background: #f7f5fb; color: #1f2430; }
    header { background: #4f46e5; color: white; padding: 28px 36px; }
    main { max-width: 1040px; margin: 28px auto; padding: 0 20px; display: grid; grid-template-columns: 340px 1fr; gap: 20px; }
    section { background: white; border: 1px solid #ded9ea; border-radius: 8px; padding: 20px; }
    h1, h2 { margin: 0 0 10px; }
    label { display: block; margin: 14px 0 6px; font-weight: 700; }
    input, select { box-sizing: border-box; width: 100%; padding: 11px; border: 1px solid #c9c3da; border-radius: 6px; }
    button { margin-top: 16px; width: 100%; padding: 12px; border: 0; border-radius: 6px; background: #4f46e5; color: white; font-weight: 700; cursor: pointer; }
    table { width: 100%; border-collapse: collapse; margin-top: 12px; }
    th, td { padding: 10px; border-bottom: 1px solid #ebe7f3; text-align: left; }
    .yes { color: #166534; font-weight: 700; }
    .no { color: #991b1b; font-weight: 700; }
    .links { margin-top: 12px; display: flex; gap: 10px; flex-wrap: wrap; }
    .links a { color: white; text-decoration: none; border: 1px solid rgba(255,255,255,.5); padding: 8px 10px; border-radius: 6px; }
    @media (max-width: 820px) { main { grid-template-columns: 1fr; } }
  </style>
</head>
<body>
  <header>
    <h1>Microservicio de Pagos</h1>
    <p>Escucha AsistenciaRegistrada, valida MongoDB por carnet y publica PagoVerificado o PagoRechazado.</p>
    <div class="links">
      <a href="http://localhost:8001">Asistencia</a>
      <a href="http://localhost:8003">Panel profesores</a>
      <a href="http://localhost:15672">RabbitMQ</a>
    </div>
  </header>
  <main>
    <section>
      <h2>Actualizar pago</h2>
      <form id="form">
        <label>Carnet</label>
        <input id="student_id" value="201544138" />
        <label>Nombre</label>
        <input id="student_name" value="Estudiante Ingenieria" />
        <label>Carrera</label>
        <select id="career">
          <option>Ciencias y Sistemas</option>
          <option>Ingenieria Civil</option>
          <option>Ingenieria Industrial</option>
        </select>
        <label>Pago</label>
        <select id="paid">
          <option value="true">Pagado</option>
          <option value="false">No pagado</option>
        </select>
        <button>Guardar en MongoDB</button>
      </form>
    </section>
    <section>
      <h2>Pagos registrados</h2>
      <table>
        <thead><tr><th>Carnet</th><th>Nombre</th><th>Carrera</th><th>Estado</th></tr></thead>
        <tbody id="rows"></tbody>
      </table>
    </section>
  </main>
  <script>
    async function loadRows() {
      const data = await fetch("/pagos").then(r => r.json());
      rows.innerHTML = data.map(x =>
        "<tr>" +
        "<td>" + x.student_id + "</td>" +
        "<td>" + (x.student_name || "") + "</td>" +
        "<td>" + (x.career || "") + "</td>" +
        "<td class='" + (x.paid ? "yes" : "no") + "'>" + (x.paid ? "PAGADO" : "NO PAGADO") + "</td>" +
        "</tr>"
      ).join("");
    }
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      await fetch("/pagos", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          student_id: student_id.value,
          carnet: student_id.value,
          student_name: student_name.value,
          career: career.value,
          paid: paid.value === "true"
        })
      });
      loadRows();
    });
    loadRows();
  </script>
</body>
</html>`;

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function connectMongo() {
  while (true) {
    try {
      const client = new MongoClient(MONGO_URL);
      await client.connect();
      payments = client.db().collection("payments");
      await payments.createIndex({ student_id: 1 }, { unique: true });
      await payments.updateOne(
        { student_id: "201544138" },
        { $set: { student_id: "201544138", carnet: "201544138", student_name: "Estudiante Ingenieria", career: "Ciencias y Sistemas", paid: true } },
        { upsert: true }
      );
      await payments.updateOne(
        { student_id: "201544139" },
        { $set: { student_id: "201544139", carnet: "201544139", student_name: "Estudiante Pendiente", career: "Ingenieria Civil", paid: false } },
        { upsert: true }
      );
      console.log("Mongo conectado");
      return;
    } catch (err) {
      console.log("Mongo no listo:", err.message);
      await sleep(2000);
    }
  }
}

async function connectRabbit() {
  while (true) {
    try {
      const conn = await amqp.connect(RABBITMQ_URL);
      channel = await conn.createChannel();
      await channel.assertExchange(EXCHANGE, "topic", { durable: true });
      console.log("RabbitMQ conectado");
      return;
    } catch (err) {
      console.log("RabbitMQ no listo:", err.message);
      await sleep(2000);
    }
  }
}

async function publish(eventType, payload) {
  channel.publish(EXCHANGE, eventType, Buffer.from(JSON.stringify(payload)), {
    contentType: "application/json",
    persistent: true
  });
}

async function consumeAttendance() {
  const queue = "pagos.attendance.created";
  await channel.assertQueue(queue, { durable: true });
  await channel.bindQueue(queue, EXCHANGE, "AsistenciaRegistrada");

  channel.consume(queue, async (msg) => {
    if (!msg) return;
    const event = JSON.parse(msg.content.toString());
    const payment = await payments.findOne({ student_id: event.carnet || event.student_id });
    const paid = Boolean(payment && payment.paid);
    const responseType = paid ? "PagoVerificado" : "PagoRechazado";

    await publish(responseType, {
      ...event,
      paid,
      checked_at: new Date().toISOString()
    });

    console.log(`${responseType} para ${event.carnet || event.student_id}`);
    channel.ack(msg);
  });

  console.log("Pagos escuchando AsistenciaRegistrada");
}

app.get("/pagos", async (req, res) => {
  const rows = await payments.find({}).toArray();
  res.json(rows);
});

app.get("/", (req, res) => {
  res.type("html").send(html);
});

app.post("/pagos", async (req, res) => {
  const { student_id, carnet, student_name, career, paid } = req.body;
  const code = carnet || student_id;
  await payments.updateOne(
    { student_id: code },
    { $set: { student_id: code, carnet: code, student_name, career, paid: Boolean(paid) } },
    { upsert: true }
  );
  res.json({ ok: true });
});

async function main() {
  await connectMongo();
  await connectRabbit();
  await consumeAttendance();
  app.listen(3000, () => console.log("Pagos service en :3000"));
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
