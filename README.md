# 🎓 Student Portal System

A full-featured web-based student information system built with **Go (Golang)** and **MySQL**.

## ✨ Features

### 👨‍💼 Admin
- User management (create, edit, delete, activate/deactivate)
- Bulk status updates
- System statistics dashboard
- Student approval management

### 🎓 Student
- Online enrollment & re-enrollment
- Payment & billing management (full pay / downpayment / installment)
- Grade viewing (GWA computation)
- Document requests (TOR, COE, Good Moral, Honorable Dismissal)
- Schedule viewing
- Lesson materials & submission uploads
- Profile management with photo upload
- Announcements feed

### 👨‍🏫 Teacher
- Class & grade management (Filipino GWA scale)
- Lesson material uploads (PDF, image, video)
- Student submission review
- Class announcements with image support

### 🏫 Registrar
- Student approval with billing assessment
- Re-enrollment application processing
- Auto-generated student ID (YYYY-NNNNN format)
- Email notifications with billing statement

### 💰 Cashier
- Payment approval (full / partial / installment)
- Installment tracking (Prelim, Midterm, Finals)

### 📁 Records Officer
- Grade release management
- Document request processing (auto-PDF generation)
- Announcement management

### 🏛️ Faculty
- Course & subject management
- Teacher scheduling & assignment
- School year management

## 🛠️ Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go (Golang) |
| Framework | Gin |
| Frontend | HTML, CSS, JavaScript |
| Database | MySQL (HeidiSQL) |
| Auth | JWT |
| Email | SMTP |
| PDF Generation | gofpdf |
| Deployment | Railway + Docker |

## ⚙️ How to Run

### Prerequisites
- Go 1.21+
- MySQL

### Steps

1. Clone the repository
```bash
git clone https://github.com/glensks/student-portal.git
cd student-portal
```

2. Configure environment
```bash
cp config/default.go config/local.go
# Edit database credentials
```

3. Run the server
```bash
go run main.go
```

## 👤 User Roles
`admin` `teacher` `student` `registrar` `cashier` `records` `faculty`

## 👨‍💻 Author
**Glen Tagubase**
- GitHub: [@glensks](https://github.com/glensks)