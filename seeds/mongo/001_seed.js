db = db.getSiblingDB("pagos_db");

db.payments.insertMany([
  { student_id: "201544138", carnet: "201544138", student_name: "Estudiante Ingenieria", career: "Ciencias y Sistemas", paid: true },
  { student_id: "201544139", carnet: "201544139", student_name: "Estudiante Pendiente", career: "Ingenieria Civil", paid: false }
]);
