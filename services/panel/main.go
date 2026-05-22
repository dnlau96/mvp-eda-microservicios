package main

import (
	"database/sql"
	"encoding/json"
	"io"
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

type TalkRating struct {
	ID       int    `json:"id"`
	TalkCode string `json:"talk_code"`
	Rating   int    `json:"rating"`
	Comment  string `json:"comment"`
	Teacher  string `json:"teacher"`
}

var db *sql.DB

var professorUser string
var professorPassword string
var professorToken string

const html = `<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Panel Profesores | MVP EDA</title>
  <style>
    body { margin: 0; font-family: Arial, sans-serif; background: #f5f8f6; color: #172018; }
    header { background: #166534; color: white; padding: 28px 36px; }
    main { max-width: 1120px; margin: 28px auto; padding: 0 20px; display: grid; grid-template-columns: 1.4fr .8fr; gap: 20px; }
    .filters { display: grid; grid-template-columns: repeat(4, minmax(160px, 1fr)); gap: 12px; align-items: end; }
    section { background: white; border: 1px solid #d8e5dc; border-radius: 8px; padding: 20px; }
    h1, h2 { margin: 0 0 10px; }
    label { display: block; margin: 14px 0 6px; font-weight: 700; }
    input, select, textarea { box-sizing: border-box; width: 100%; padding: 11px; border: 1px solid #bdd0c3; border-radius: 6px; }
    textarea { min-height: 72px; resize: vertical; }
    button { width: 100%; padding: 12px; border: 0; border-radius: 6px; background: #166534; color: white; font-weight: 700; cursor: pointer; }
    button.danger { background: #b91c1c; }
    .row-actions { display: flex; gap: 8px; }
    .row-actions button { width: auto; padding: 8px 10px; }
    table { width: 100%; border-collapse: collapse; margin-top: 12px; }
    th, td { padding: 10px; border-bottom: 1px solid #e5eee8; text-align: left; }
    .tag { display: inline-block; padding: 4px 8px; border-radius: 999px; background: #dcfce7; color: #166534; font-size: 12px; font-weight: 700; }
    .links { margin-top: 12px; display: flex; gap: 10px; flex-wrap: wrap; }
    .links a { color: white; text-decoration: none; border: 1px solid rgba(255,255,255,.5); padding: 8px 10px; border-radius: 6px; }
    .logout { margin-top: 12px; }
    .logout button { width: auto; border: 1px solid rgba(255,255,255,.5); background: transparent; padding: 8px 10px; }
    @media (max-width: 880px) { main, .filters { grid-template-columns: 1fr; } }
  </style>
</head>
<body>
  <header>
    <h1>Panel de Profesores y Calidad</h1>
    <p>Consulta asistentes aprobados por charla, carrera, carnet o nombre para manejo de grupos grandes.</p>
    <div class="links">
      <a href="http://localhost:8001">Asistencia</a>
      <a href="http://localhost:8002">Pagos</a>
      <a href="http://localhost:15672">RabbitMQ</a>
    </div>
    <form class="logout" method="post" action="/logout">
      <button>Cerrar sesion</button>
    </form>
  </header>
  <main>
    <section>
      <h2>Filtros</h2>
      <form id="filters" class="filters">
        <div>
          <label>Codigo de charla</label>
          <select id="talk_code"><option value="">Todas</option></select>
        </div>
        <div>
          <label>Carrera</label>
          <select id="career">
            <option value="">Todas</option>
            <option>Ciencias y Sistemas</option>
            <option>Ingenieria Civil</option>
            <option>Ingenieria Industrial</option>
          </select>
        </div>
        <div>
          <label>Buscar carnet o nombre</label>
          <input id="q" placeholder="201544138 o nombre" />
        </div>
        <div>
          <label>Limite</label>
          <select id="limit">
            <option>50</option>
            <option>100</option>
            <option>200</option>
            <option>400</option>
          </select>
        </div>
        <button>Buscar asistentes</button>
      </form>
      <h2 style="margin-top:24px">Estudiantes aprobados</h2>
      <table>
        <thead><tr><th>Asistencia</th><th>Carnet</th><th>Carrera</th><th>Charla</th><th>Estado</th><th>Acciones</th></tr></thead>
        <tbody id="approved"></tbody>
      </table>
    </section>
    <section>
      <h2>Calificar charla</h2>
      <form id="ratingForm">
        <label>Profesor</label>
        <input id="teacher" value="Profesor Demo" />
        <input id="rating_id" type="hidden" />
        <label>Charla</label>
        <select id="rating_talk_code"></select>
        <label>Puntuacion</label>
        <select id="rating_value">
          <option value="5">5 - Excelente</option>
          <option value="4">4 - Muy buena</option>
          <option value="3">3 - Buena</option>
          <option value="2">2 - Regular</option>
          <option value="1">1 - Mala</option>
        </select>
        <label>Comentario</label>
        <textarea id="rating_comment">Buena charla para el congreso</textarea>
        <button>Guardar calificacion</button>
      </form>
      <h2 style="margin-top:24px">Calificaciones de profesores</h2>
      <table>
        <thead><tr><th>Charla</th><th>Profesor</th><th>Nota</th><th>Acciones</th></tr></thead>
        <tbody id="ratingRows"></tbody>
      </table>
    </section>
  </main>
  <script>
    async function loadTalks() {
      const data = await fetch("/charlas").then(r => r.json());
      talk_code.innerHTML = "<option value=''>Todas</option>" + data.map(x =>
        "<option value='" + x.talk_code + "'>" + x.talk_code + " - " + x.talk_title + "</option>"
      ).join("");
      rating_talk_code.innerHTML = data.map(x =>
        "<option value='" + x.talk_code + "'>" + x.talk_code + " - " + x.talk_title + "</option>"
      ).join("");
    }
    async function loadApproved() {
      const params = new URLSearchParams();
      if (talk_code.value) params.set("talk_code", talk_code.value);
      if (career.value) params.set("career", career.value);
      if (q.value) params.set("q", q.value);
      params.set("limit", limit.value);
      const data = await fetch("/panel?" + params.toString()).then(r => r.json());
      approved.innerHTML = data.map(x =>
        "<tr>" +
        "<td>#" + x.attendance_id + "</td>" +
        "<td>" + x.student_name + "<br><small>" + x.student_id + "</small></td>" +
        "<td>" + x.career + "</td>" +
        "<td>" + x.talk_id + "<br><small>" + x.talk_title + "</small></td>" +
        "<td><span class='tag'>APROBADO</span></td>" +
        "<td><div class='row-actions'><button type='button' class='danger' onclick='deleteApproved(" + x.attendance_id + ")'>Eliminar</button></div></td>" +
        "</tr>"
      ).join("");
    }
    filters.addEventListener("submit", async (event) => {
      event.preventDefault();
      loadApproved();
    });
    async function loadRatings() {
      const data = await fetch("/calificaciones-charla").then(r => r.json());
      ratingRows.innerHTML = data.map(x =>
        "<tr>" +
        "<td>" + x.talk_code + "</td>" +
        "<td>" + x.teacher + "</td>" +
        "<td>" + x.rating + "/5</td>" +
        "<td><div class='row-actions'>" +
        "<button type='button' onclick='editRating(" + JSON.stringify(x) + ")'>Editar</button>" +
        "<button type='button' class='danger' onclick='deleteRating(" + x.id + ")'>Eliminar</button>" +
        "</div></td>" +
        "</tr>"
      ).join("");
    }
    function editRating(row) {
      rating_id.value = row.id || "";
      rating_talk_code.value = row.talk_code;
      teacher.value = row.teacher;
      rating_value.value = String(row.rating);
      rating_comment.value = row.comment || "";
    }
    async function deleteRating(id) {
      if (!confirm("Eliminar calificacion?")) return;
      await fetch("/calificaciones-charla/" + id, { method: "DELETE" });
      loadRatings();
    }
    async function deleteApproved(attendanceId) {
      if (!confirm("Eliminar registro aprobado #" + attendanceId + " del panel?")) return;
      await fetch("/panel/" + attendanceId, { method: "DELETE" });
      loadApproved();
    }
    ratingForm.addEventListener("submit", async (event) => {
      event.preventDefault();
      await fetch("/calificar-charla", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          talk_code: rating_talk_code.value,
          id: Number(rating_id.value || 0),
          rating: Number(rating_value.value),
          comment: rating_comment.value,
          teacher: teacher.value
        })
      });
      rating_id.value = "";
      loadRatings();
    });
    loadTalks().then(loadApproved);
    loadRatings();
    setInterval(loadApproved, 2500);
  </script>
</body>
</html>`

const loginHTML = `<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Login Profesores</title>
  <style>
    body { margin: 0; min-height: 100vh; display: grid; place-items: center; font-family: Arial, sans-serif; background: #f5f8f6; color: #172018; }
    section { width: min(380px, calc(100vw - 32px)); background: white; border: 1px solid #d8e5dc; border-radius: 8px; padding: 24px; }
    h1 { margin: 0 0 8px; }
    label { display: block; margin: 14px 0 6px; font-weight: 700; }
    input { box-sizing: border-box; width: 100%; padding: 11px; border: 1px solid #bdd0c3; border-radius: 6px; }
    button { margin-top: 16px; width: 100%; padding: 12px; border: 0; border-radius: 6px; background: #166534; color: white; font-weight: 700; cursor: pointer; }
    .hint { color: #475569; }
  </style>
</head>
<body>
  <section>
    <h1>Panel de profesores</h1>
    <p class="hint">Inicia sesion para consultar asistentes y calificar charlas.</p>
    <form method="post" action="/login">
      <label>Usuario</label>
      <input name="username" />
      <label>Password</label>
      <input name="password" type="password" />
      <button>Entrar</button>
    </form>
  </section>
</body>
</html>`

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("profesor_session")
	return err == nil && cookie.Value == professorToken
}

func requireAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAuthenticated(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{"error": "No autorizado"})
			return
		}
		handler(w, r)
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if r.FormValue("username") != professorUser || r.FormValue("password") != professorPassword {
		http.Error(w, "Credenciales invalidas", http.StatusUnauthorized)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "profesor_session",
		Value:    professorToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "profesor_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
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
		`CREATE TABLE IF NOT EXISTS professor_talk_ratings (
			id INT AUTO_INCREMENT PRIMARY KEY,
			talk_id VARCHAR(50) NOT NULL,
			teacher VARCHAR(120) NOT NULL,
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
	query := `SELECT a.attendance_id, a.student_id, a.student_name, a.career, a.talk_id, a.talk_title, a.created_at
		FROM approved_attendance a
		INNER JOIN (
			SELECT MAX(id) AS id
			FROM approved_attendance
			GROUP BY student_id, talk_id
		) latest ON latest.id = a.id
		WHERE 1=1`
	args := []any{}
	params := r.URL.Query()
	if talkCode := params.Get("talk_code"); talkCode != "" {
		query += " AND a.talk_id = ?"
		args = append(args, talkCode)
	}
	if career := params.Get("career"); career != "" {
		query += " AND a.career = ?"
		args = append(args, career)
	}
	if q := params.Get("q"); q != "" {
		query += " AND (a.student_id LIKE ? OR a.student_name LIKE ?)"
		like := "%" + q + "%"
		args = append(args, like, like)
	}
	limit := params.Get("limit")
	if limit != "100" && limit != "200" && limit != "400" {
		limit = "50"
	}
	query += " ORDER BY a.id DESC LIMIT " + limit

	rows, err := db.Query(query, args...)
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

func rateTalkHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}
	var rating TalkRating
	if err := json.NewDecoder(r.Body).Decode(&rating); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if rating.Rating < 1 || rating.Rating > 5 {
		http.Error(w, "rating debe estar entre 1 y 5", 400)
		return
	}
	if rating.Teacher == "" {
		rating.Teacher = "Profesor Demo"
	}
	var err error
	if rating.ID > 0 {
		_, err = db.Exec(
			"UPDATE professor_talk_ratings SET talk_id = ?, teacher = ?, rating = ?, comment = ?, created_at = CURRENT_TIMESTAMP WHERE id = ?",
			rating.TalkCode, rating.Teacher, rating.Rating, rating.Comment, rating.ID,
		)
	} else {
		_, err = db.Exec(
			"INSERT INTO professor_talk_ratings (talk_id, teacher, rating, comment) VALUES (?, ?, ?, ?)",
			rating.TalkCode, rating.Teacher, rating.Rating, rating.Comment,
		)
	}
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func talkRatingsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, talk_id, teacher, rating, comment, created_at FROM professor_talk_ratings ORDER BY id DESC LIMIT 100")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	result := []map[string]any{}
	for rows.Next() {
		var id int
		var talkID, teacher, comment string
		var rating int
		var createdAt time.Time
		rows.Scan(&id, &talkID, &teacher, &rating, &comment, &createdAt)
		result = append(result, map[string]any{
			"id":         id,
			"talk_code":  talkID,
			"talk_id":    talkID,
			"teacher":    teacher,
			"rating":     rating,
			"comment":    comment,
			"created_at": createdAt.Format(time.RFC3339),
		})
	}
	json.NewEncoder(w).Encode(result)
}

func deleteTalkRatingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "use DELETE", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Path[len("/calificaciones-charla/"):]
	_, err := db.Exec("DELETE FROM professor_talk_ratings WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func deleteApprovedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "use DELETE", http.StatusMethodNotAllowed)
		return
	}
	attendanceID := r.URL.Path[len("/panel/"):]
	_, err := db.Exec("DELETE FROM approved_attendance WHERE attendance_id = ?", attendanceID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func talksProxyHandler(w http.ResponseWriter, r *http.Request) {
	asistenciaURL := env("ASISTENCIA_URL", "http://asistencia-service:8000")
	resp, err := http.Get(asistenciaURL + "/charlas")
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if !isAuthenticated(r) {
		w.Write([]byte(loginHTML))
		return
	}
	w.Write([]byte(html))
}

func main() {
	dsn := env("MYSQL_DSN", "panel:panel@tcp(localhost:3306)/panel_db?parseTime=true")
	rabbitURL := env("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	professorUser = env("PROFESORES_USER", "profesor")
	professorPassword = env("PROFESORES_PASSWORD", "profesor123")
	professorToken = env("PROFESORES_TOKEN", "profesores-demo-token")

	waitForMySQL(dsn)
	initDB()
	go consumeVerifiedPayments(rabbitURL)

	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/panel", requireAuth(panelHandler))
	http.HandleFunc("/panel/", requireAuth(deleteApprovedHandler))
	http.HandleFunc("/calificar", requireAuth(rateHandler))
	http.HandleFunc("/calificaciones", requireAuth(ratingsHandler))
	http.HandleFunc("/calificar-charla", requireAuth(rateTalkHandler))
	http.HandleFunc("/calificaciones-charla", requireAuth(talkRatingsHandler))
	http.HandleFunc("/calificaciones-charla/", requireAuth(deleteTalkRatingHandler))
	http.HandleFunc("/charlas", requireAuth(talksProxyHandler))
	http.HandleFunc("/", homeHandler)

	log.Println("Panel service en :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
