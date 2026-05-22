CREATE TABLE IF NOT EXISTS attendance (
  id SERIAL PRIMARY KEY,
  student_id VARCHAR(50) NOT NULL,
  student_name VARCHAR(120) NOT NULL,
  career VARCHAR(80) NOT NULL DEFAULT 'Ciencias y Sistemas',
  talk_id VARCHAR(50) NOT NULL,
  talk_title VARCHAR(160) NOT NULL DEFAULT 'Arquitecturas Dirigidas por Eventos',
  status VARCHAR(20) NOT NULL DEFAULT 'PENDIENTE',
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_attendance_student_talk
ON attendance (student_id, talk_id);

CREATE TABLE IF NOT EXISTS talk_ratings (
  id SERIAL PRIMARY KEY,
  student_id VARCHAR(50) NOT NULL,
  talk_id VARCHAR(50) NOT NULL,
  rating INT NOT NULL CHECK (rating BETWEEN 1 AND 5),
  comment VARCHAR(255),
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_talk_ratings_student_talk
ON talk_ratings (student_id, talk_id);
