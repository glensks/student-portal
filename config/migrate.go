package config

import "log"

func MigrateDB() {

	// ----------------  DROP ALL TABLES (reset) ----------------
	drops := []string{
		`SET FOREIGN_KEY_CHECKS = 0`,
		`DROP TABLE IF EXISTS password_reset_tokens`,
		`DROP TABLE IF EXISTS student_submissions`,
		`DROP TABLE IF EXISTS lesson_materials`,
		`DROP TABLE IF EXISTS records_announcements`,
		`DROP TABLE IF EXISTS registrar_announcements`,
		`DROP TABLE IF EXISTS announcements`,
		`DROP TABLE IF EXISTS enrollment_applications`,
		`DROP TABLE IF EXISTS document_requests`,
		`DROP TABLE IF EXISTS student_installments`,
		`DROP TABLE IF EXISTS payment_fees`,
		`DROP TABLE IF EXISTS student_payments`,
		`DROP TABLE IF EXISTS grades`,
		`DROP TABLE IF EXISTS school_year`,
		`DROP TABLE IF EXISTS teacher_subjects`,
		`DROP TABLE IF EXISTS subjects`,
		`DROP TABLE IF EXISTS courses`,
		`DROP TABLE IF EXISTS student_academic`,
		`DROP TABLE IF EXISTS student_family`,
		`DROP TABLE IF EXISTS students`,
		`DROP TABLE IF EXISTS users`,
		`SET FOREIGN_KEY_CHECKS = 1`,
	}

	for _, q := range drops {
		DB.Exec(q)
	}

	log.Println("🗑️ Old tables dropped")

	// ---------------- CREATE ALL TABLES ----------------
	queries := []string{

		`CREATE TABLE IF NOT EXISTS users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			first_name VARCHAR(255),
			middle_name VARCHAR(255),
			surname VARCHAR(255),
			email VARCHAR(255) UNIQUE,
			contact_number VARCHAR(50),
			role VARCHAR(50) DEFAULT 'student',
			status VARCHAR(50) DEFAULT 'active',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS students (
			id INT AUTO_INCREMENT PRIMARY KEY,
			student_id VARCHAR(100) UNIQUE NOT NULL,
			password VARCHAR(255),
			first_name VARCHAR(255),
			middle_name VARCHAR(255),
			last_name VARCHAR(255),
			age INT DEFAULT 18,
			contact_number VARCHAR(50),
			email VARCHAR(255),
			address TEXT,
			status VARCHAR(50) DEFAULT 'pending',
			profile_picture VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS student_family (
			id INT AUTO_INCREMENT PRIMARY KEY,
			student_id INT NOT NULL,
			father_first_name VARCHAR(255),
			father_middle_name VARCHAR(255),
			father_last_name VARCHAR(255),
			father_occupation VARCHAR(255),
			father_contact_number VARCHAR(50),
			father_address TEXT,
			mother_first_name VARCHAR(255),
			mother_middle_name VARCHAR(255),
			mother_last_name VARCHAR(255),
			mother_occupation VARCHAR(255),
			mother_contact_number VARCHAR(50),
			mother_address TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (student_id) REFERENCES students(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS student_academic (
			id INT AUTO_INCREMENT PRIMARY KEY,
			student_id INT NOT NULL,
			last_school_attended VARCHAR(255),
			last_school_year VARCHAR(50),
			course INT,
			subjects TEXT,
			year_level VARCHAR(50) DEFAULT '1',
			semester VARCHAR(50) DEFAULT '1st',
			scholarship_status VARCHAR(100) DEFAULT 'non-scholar',
			total_units INT DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (student_id) REFERENCES students(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS courses (
			id INT AUTO_INCREMENT PRIMARY KEY,
			course_name VARCHAR(255) NOT NULL,
			code VARCHAR(100) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS subjects (
			id INT AUTO_INCREMENT PRIMARY KEY,
			subject_name VARCHAR(255) NOT NULL,
			code VARCHAR(100),
			course_id INT,
			year_level VARCHAR(50),
			semester VARCHAR(50),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS teacher_subjects (
			id INT AUTO_INCREMENT PRIMARY KEY,
			teacher_id INT NOT NULL,
			subject_id INT NOT NULL,
			course_id INT NOT NULL,
			room VARCHAR(100),
			day VARCHAR(50),
			time_start VARCHAR(50),
			time_end VARCHAR(50),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (teacher_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (subject_id) REFERENCES subjects(id) ON DELETE CASCADE,
			FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS school_year (
			id INT AUTO_INCREMENT PRIMARY KEY,
			year VARCHAR(50) NOT NULL,
			semester VARCHAR(50) NOT NULL,
			is_active BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS grades (
			id INT AUTO_INCREMENT PRIMARY KEY,
			student_id INT NOT NULL,
			subject_id INT NOT NULL,
			teacher_id INT NOT NULL,
			prelim FLOAT,
			midterm FLOAT,
			finals FLOAT,
			remarks TEXT,
			is_released BOOLEAN DEFAULT FALSE,
			released_at TIMESTAMP NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (student_id) REFERENCES students(id) ON DELETE CASCADE,
			FOREIGN KEY (subject_id) REFERENCES subjects(id) ON DELETE CASCADE,
			FOREIGN KEY (teacher_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS student_payments (
			id INT AUTO_INCREMENT PRIMARY KEY,
			student_id INT NOT NULL,
			total_amount INT DEFAULT 0,
			amount_paid INT DEFAULT 0,
			downpayment_amount FLOAT DEFAULT 0,
			payment_method VARCHAR(100),
			semester VARCHAR(50),
			school_year VARCHAR(50),
			status VARCHAR(50) DEFAULT 'unpaid',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (student_id) REFERENCES students(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS payment_fees (
			id INT AUTO_INCREMENT PRIMARY KEY,
			payment_id INT NOT NULL,
			fee_name VARCHAR(255) NOT NULL,
			amount INT DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (payment_id) REFERENCES student_payments(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS student_installments (
			id INT AUTO_INCREMENT PRIMARY KEY,
			payment_id INT NOT NULL,
			term VARCHAR(50),
			amount FLOAT DEFAULT 0,
			status VARCHAR(50) DEFAULT 'unpaid',
			paid_at TIMESTAMP NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (payment_id) REFERENCES student_payments(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS document_requests (
			id INT AUTO_INCREMENT PRIMARY KEY,
			student_id INT NOT NULL,
			document_type VARCHAR(255) NOT NULL,
			purpose TEXT,
			copies INT DEFAULT 1,
			status VARCHAR(50) DEFAULT 'pending',
			notes TEXT,
			document_file VARCHAR(255),
			requested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			processed_at TIMESTAMP NULL,
			FOREIGN KEY (student_id) REFERENCES students(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS enrollment_applications (
			id INT AUTO_INCREMENT PRIMARY KEY,
			student_id INT NOT NULL,
			academic_year VARCHAR(50),
			year_level INT DEFAULT 1,
			semester VARCHAR(50),
			course_id INT,
			subjects TEXT,
			total_units INT DEFAULT 0,
			scholarship_status VARCHAR(100) DEFAULT 'non-scholar',
			status VARCHAR(50) DEFAULT 'pending',
			remarks TEXT,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			processed_at TIMESTAMP NULL,
			FOREIGN KEY (student_id) REFERENCES students(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS lesson_materials (
			id INT AUTO_INCREMENT PRIMARY KEY,
			teacher_id INT NOT NULL,
			subject_id INT NOT NULL,
			class_id INT,
			title VARCHAR(255) NOT NULL,
			description TEXT,
			type VARCHAR(50),
			file_name VARCHAR(255),
			file_path VARCHAR(255),
			file_size BIGINT,
			due_date TIMESTAMP NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (teacher_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (subject_id) REFERENCES subjects(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS student_submissions (
			id INT AUTO_INCREMENT PRIMARY KEY,
			student_id INT NOT NULL,
			material_id INT NOT NULL,
			file_name VARCHAR(255),
			file_path VARCHAR(255),
			file_size BIGINT,
			status VARCHAR(50) DEFAULT 'on-time',
			remarks TEXT,
			submitted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			reviewed_at TIMESTAMP NULL,
			FOREIGN KEY (student_id) REFERENCES students(id) ON DELETE CASCADE,
			FOREIGN KEY (material_id) REFERENCES lesson_materials(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS announcements (
			id INT AUTO_INCREMENT PRIMARY KEY,
			teacher_id INT NOT NULL,
			class_id INT,
			subject_id INT,
			title VARCHAR(255) NOT NULL,
			content TEXT,
			image_name VARCHAR(255),
			image_path VARCHAR(255),
			image_size BIGINT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (teacher_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS registrar_announcements (
			id INT AUTO_INCREMENT PRIMARY KEY,
			registrar_id INT NOT NULL,
			title VARCHAR(255) NOT NULL,
			content TEXT,
			target_audience VARCHAR(50) DEFAULT 'all',
			image_name VARCHAR(255),
			image_path VARCHAR(255),
			image_size BIGINT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (registrar_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS records_announcements (
			id INT AUTO_INCREMENT PRIMARY KEY,
			records_officer_id INT NOT NULL,
			title VARCHAR(255) NOT NULL,
			content TEXT,
			image_name VARCHAR(255),
			image_path VARCHAR(255),
			image_size BIGINT,
			priority VARCHAR(50) DEFAULT 'normal',
			target_audience VARCHAR(50) DEFAULT 'all',
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (records_officer_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS password_reset_tokens (
			id INT AUTO_INCREMENT PRIMARY KEY,
			student_id INT NOT NULL,
			token VARCHAR(255) UNIQUE NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			used TINYINT DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (student_id) REFERENCES students(id) ON DELETE CASCADE
		)`,
	}

	for _, query := range queries {
		_, err := DB.Exec(query)
		if err != nil {
			log.Fatal("Migration error: ", err)
		}
	}

	log.Println("✅ Database migrated successfully")
}
