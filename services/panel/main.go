package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	amqp "github.com/rabbitmq/amqp091-go"
)

const exchange = "mvp_eventos"

type PaymentVerified struct {
	AttendanceID int    `json:"attendance_id"`
	StudentID    string `json:"student_id"`
	Carnet       string `json:"carnet"`
	StudentName  string `json:"student_name"`
	Career       string `json:"career"`
	TalkID       string `json:"talk_id"`
	TalkCode     string `json:"talk_code"`
	TalkTitle    string `json:"talk_title"`
}

type Rating struct {
	StudentID    string `json:"student_id"`
	Carnet       string `json:"carnet"`
	TalkID       string `json:"talk_id"`
	TalkCode     string `json:"talk_code"`
	ReviewerType string `json:"reviewer_type"`
	Rating       int    `json:"rating"`
	Comment      string `json:"comment"`
}

var db *sql.DB

const html = `<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Panel Profesores | MVP EDA</title>
  <style>
    body { margin: 0; font-family: Arial, sans-serif; background: #f5f8f6; color: #172018; }
    header { background: #166534; color: white; padding: 28px 36px; }
    main { max-width: 1120px; margin: 28px auto; padding: 0 20px; display: grid; grid-template-columns: 1.2fr .8fr; gap: 20px; }
    section { background: white; border: 1px solid #d8e5dc; border-radius: 8px; padding: 20px; }
    h1, h2 { margin: 0 0 10px; }
    label { display: block; margin: 14px 0 6px; font-weight: 700; }
    input, textarea { box-sizing: border-box; width: 100%; padding: 11px; border: 1px solid #bdd0c3; border-radius: 6px; }
    textarea { min-height: 72px; resize: vertical; }
    button { margin-top: 16px; width: 100%; padding: 12px; border: 0; border-radius: 6px; background: #166534; color: white; font-weight: 700; cursor: pointer; }
    table { width: 100%; border-collapse: collapse; margin-top: 12px; }
    th, td { padding: 10px; border-bottom: 1px solid #e5eee8; text-align: left; }
    .tag { display: inline-block; padding: 4px 8px; border-radius: 999px; background: #dcfce7; color: #166534; font-size: 12px; font-weight: 700; }
    .links { margin-top: 12px; display: flex; gap: 10px; flex-wrap: wrap; }
    .links a { color: white; text-decoration: none; border: 1px solid rgba(255,255,255,.5); padding: 8px 10px; border-radius: 6px; }
    @media (max-width: 880px) { main { grid-template-columns: 1fr; } }
  </style>
</head>
<body>
  <header>
    <h1>Panel de Profesores y Calidad</h1>
    <p>Recibe PagoVerificado, muestra asistentes aprobados y registra calificaciones solo si el carnet asistio a esa charla.</p>
    <div class="links">
      <a href="http://localhost:8001">Asistencia</a>
      <a href="http://localhost:8002">Pagos</a>
      <a href="http://localhost:15672">RabbitMQ</a>
    </div>
  </header>
  <main>
    <section>
      <h2>Estudiantes aprobados en la charla</h2>
      <table>
        <thead><tr><th>Asistencia</th><th>Carnet</th><th>Carrera</th><th>Charla</th><th>Estado</th></tr></thead>
        <tbody id="approved"></tbody>
      </table>
    </section>
    <section>
      <h2>Calificar charla</h2>
      <form id="form">
        <label>Carnet</label>
        <input id="student_id" value="201544138" />
        <label>Codigo de charla</label>
        <input id="talk_id" value="SIS-EDA-001" />
        <label>Quien califica</label>
        <input id="reviewer_type" list="reviewers" value="ESTUDIANTE" />
        <datalist id="reviewers">
          <option value="ESTUDIANTE"></option>
          <option value="PROFESOR"></option>
        </datalist>
        <label>Nota</label>
        <input id="rating" type="number" min="1" max="5" value="5" />
        <label>Comentario</label>
        <textarea id="comment">Excelente</textarea>
        <button>Guardar calificacion</button>
      </form>
      <h2 style="margin-top:24px">Ultimas calificaciones</h2>
      <table>
        <thead><tr><th>Carnet</th><th>Tipo</th><th>Nota</th><th>Comentario</th></tr></thead>
        <tbody id="ratings"></tbody>
      </table>
    </section>
  </main>
  <script>
    async function loadApproved() {
      const data = await fetch("/panel").then(r => r.json());
      approved.innerHTML = data.map(x =>
        "<tr>" +
        "<td>#" + x.attendance_id + "</td>" +
        "<td>" + x.student_name + "<br><small>" + x.student_id + "</small></td>" +
        "<td>" + x.career + "</td>" +
        "<td>" + x.talk_id + "<br><small>" + x.talk_title + "</small></td>" +
        "<td><span class='tag'>APROBADO</span></td>" +
        "</tr>"
      ).join("");
    }
    async function loadRatings() {
      const data = await fetch("/calificaciones").then(r => r.json());
      ratings.innerHTML = data.map(x =>
        "<tr>" +
        "<td>" + x.student_id + "</td>" +
        "<td>" + x.reviewer_type + "</td>" +
        "<td>" + x.rating + "/5</td>" +
        "<td>" + (x.comment || "") + "</td>" +
        "</tr>"
      ).join("");
    }
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      await fetch("/calificar", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          student_id: student_id.value,
          carnet: student_id.value,
          talk_id: talk_id.value,
          talk_code: talk_id.value,
          reviewer_type: reviewer_type.value,
          rating: Number(rating.value),
          comment: comment.value
        })
      });
      loadRatings();
    });
    loadApproved();
    loadRatings();
    setInterval(loadApproved, 2500);
  </script>
</body>
</html>`

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func waitForMySQL(dsn string) {
	for {
		conn, err := sql.Open("mysql", dsn)
		if err == nil && conn.Ping() == nil {
			db = conn
			return
		}
		if conn != nil {
			conn.Close()
		}
		log.Println("MySQL no listo:", err)
		time.Sleep(2 * time.Second)
	}
}

func initDB() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS approved_attendance (
			id INT AUTO_INCREMENT PRIMARY KEY,
			attendance_id INT NOT NULL,
			student_id VARCHAR(50) NOT NULL,
			student_name VARCHAR(120) NOT NULL,
			career VARCHAR(80) NOT NULL DEFAULT 'Ciencias y Sistemas',
			talk_id VARCHAR(50) NOT NULL,
			talk_title VARCHAR(160) NOT NULL DEFAULT 'Arquitecturas Dirigidas por Eventos',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS ratings (
			id INT AUTO_INCREMENT PRIMARY KEY,
			student_id VARCHAR(50) NOT NULL,
			talk_id VARCHAR(50) NOT NULL,
			reviewer_type VARCHAR(20) NOT NULL DEFAULT 'ESTUDIANTE',
			rating INT NOT NULL,
			comment VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			log.Fatal(err)
		}
	}
	alterQueries := []string{
		`ALTER TABLE approved_attendance ADD COLUMN career VARCHAR(80) NOT NULL DEFAULT 'Ciencias y Sistemas'`,
		`ALTER TABLE approved_attendance ADD COLUMN talk_title VARCHAR(160) NOT NULL DEFAULT 'Arquitecturas Dirigidas por Eventos'`,
		`ALTER TABLE ratings ADD COLUMN reviewer_type VARCHAR(20) NOT NULL DEFAULT 'ESTUDIANTE'`,
	}
	for _, query := range alterQueries {
		db.Exec(query)
	}
}

func connectRabbit(url string) (*amqp.Connection, *amqp.Channel) {
	for {
		conn, err := amqp.Dial(url)
		if err == nil {
			ch, err := conn.Channel()
			if err == nil {
				err = ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil)
				if err == nil {
					return conn, ch
				}
			}
		}
		log.Println("RabbitMQ no listo:", err)
		time.Sleep(2 * time.Second)
	}
}

func consumeVerifiedPayments(rabbitURL string) {
	for {
		conn, ch := connectRabbit(rabbitURL)
		queue, err := ch.QueueDeclare("panel.payment.verified", true, false, false, false, nil)
		if err != nil {
			log.Println(err)
			time.Sleep(2 * time.Second)
			continue
		}
		ch.QueueBind(queue.Name, "PagoVerificado", exchange, false, nil)
		msgs, err := ch.Consume(queue.Name, "", false, false, false, false, nil)
		if err != nil {
			log.Println(err)
			conn.Close()
			time.Sleep(2 * time.Second)
			continue
		}

		log.Println("Panel escuchando PagoVerificado")
		for msg := range msgs {
			var event PaymentVerified
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				log.Println(err)
				msg.Nack(false, false)
				continue
			}
			carnet := event.Carnet
			if carnet == "" {
				carnet = event.StudentID
			}
			talkCode := event.TalkCode
			if talkCode == "" {
				talkCode = event.TalkID
			}
			_, err := db.Exec(
				`INSERT INTO approved_attendance (attendance_id, student_id, student_name, career, talk_id, talk_title)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				event.AttendanceID, carnet, event.StudentName, event.Career, talkCode, event.TalkTitle,
			)
			if err != nil {
				log.Println(err)
				msg.Nack(false, true)
				continue
			}
			log.Println("Panel actualizado con", carnet)
			msg.Ack(false)
		}
		conn.Close()
		time.Sleep(2 * time.Second)
	}
}

func panelHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT attendance_id, student_id, student_name, career, talk_id, talk_title, created_at
		FROM approved_attendance ORDER BY id DESC`)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	result := []map[string]any{}
	for rows.Next() {
		var attendanceID int
		var studentID, studentName, career, talkID, talkTitle string
		var createdAt time.Time
		rows.Scan(&attendanceID, &studentID, &studentName, &career, &talkID, &talkTitle, &createdAt)
		result = append(result, map[string]any{
			"attendance_id": attendanceID,
			"student_id":    studentID,
			"carnet":        studentID,
			"student_name":  studentName,
			"career":        career,
			"talk_id":       talkID,
			"talk_code":     talkID,
			"talk_title":    talkTitle,
			"created_at":    createdAt.Format(time.RFC3339),
		})
	}
	json.NewEncoder(w).Encode(result)
}

func rateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}
	var rating Rating
	if err := json.NewDecoder(r.Body).Decode(&rating); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	carnet := rating.Carnet
	if carnet == "" {
		carnet = rating.StudentID
	}
	talkCode := rating.TalkCode
	if talkCode == "" {
		talkCode = rating.TalkID
	}
	reviewerType := rating.ReviewerType
	if reviewerType == "" {
		reviewerType = "ESTUDIANTE"
	}
	var exists int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM approved_attendance WHERE student_id = ? AND talk_id = ?",
		carnet, talkCode,
	).Scan(&exists)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if exists == 0 {
		http.Error(w, "No se puede calificar: el carnet no tiene asistencia aprobada para esa charla", http.StatusBadRequest)
		return
	}
	_, err = db.Exec(
		"INSERT INTO ratings (student_id, talk_id, reviewer_type, rating, comment) VALUES (?, ?, ?, ?, ?)",
		carnet, talkCode, reviewerType, rating.Rating, rating.Comment,
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func ratingsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT student_id, talk_id, reviewer_type, rating, comment, created_at FROM ratings ORDER BY id DESC")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	result := []map[string]any{}
	for rows.Next() {
		var studentID, talkID, reviewerType, comment string
		var rating int
		var createdAt time.Time
		rows.Scan(&studentID, &talkID, &reviewerType, &rating, &comment, &createdAt)
		result = append(result, map[string]any{
			"student_id":    studentID,
			"carnet":        studentID,
			"talk_id":       talkID,
			"talk_code":     talkID,
			"reviewer_type": reviewerType,
			"rating":        rating,
			"comment":       comment,
			"created_at":    createdAt.Format(time.RFC3339),
		})
	}
	json.NewEncoder(w).Encode(result)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func main() {
	dsn := env("MYSQL_DSN", "panel:panel@tcp(localhost:3306)/panel_db?parseTime=true")
	rabbitURL := env("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")

	waitForMySQL(dsn)
	initDB()
	go consumeVerifiedPayments(rabbitURL)

	http.HandleFunc("/panel", panelHandler)
	http.HandleFunc("/calificar", rateHandler)
	http.HandleFunc("/calificaciones", ratingsHandler)
	http.HandleFunc("/", homeHandler)

	log.Println("Panel service en :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
