-- phpMyAdmin SQL Dump
-- version 5.2.1
-- https://www.phpmyadmin.net/
--
-- Host: 127.0.0.1
-- Generation Time: Feb 21, 2026 at 05:15 AM
-- Server version: 10.4.32-MariaDB
-- PHP Version: 8.2.12

SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
START TRANSACTION;
SET time_zone = "+00:00";


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;

--
-- Database: `student-portal`
--

-- --------------------------------------------------------

--
-- Table structure for table `announcements`
--

CREATE TABLE `announcements` (
  `id` int(11) NOT NULL,
  `teacher_id` int(11) NOT NULL,
  `class_id` int(11) NOT NULL,
  `subject_id` int(11) NOT NULL,
  `title` varchar(255) NOT NULL,
  `content` text NOT NULL,
  `image_name` varchar(255) DEFAULT NULL,
  `image_path` varchar(500) DEFAULT NULL,
  `image_size` bigint(20) DEFAULT NULL,
  `created_at` datetime DEFAULT current_timestamp(),
  `updated_at` datetime DEFAULT current_timestamp() ON UPDATE current_timestamp()
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- Table structure for table `class_schedules`
--

CREATE TABLE `class_schedules` (
  `id` int(11) NOT NULL,
  `subject_id` int(11) NOT NULL,
  `day_of_week` varchar(20) NOT NULL,
  `start_time` time NOT NULL,
  `end_time` time NOT NULL,
  `room` varchar(50) NOT NULL,
  `instructor` varchar(100) DEFAULT NULL,
  `semester` varchar(20) DEFAULT NULL,
  `school_year` varchar(20) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp()
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- Table structure for table `courses`
--

CREATE TABLE `courses` (
  `id` int(11) NOT NULL,
  `course_name` varchar(255) NOT NULL,
  `code` varchar(50) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `courses`
--

INSERT INTO `courses` (`id`, `course_name`, `code`) VALUES
(44, 'BACHELOR OF SCIENCE IN COMPUTER SCIENCE', 'BSCS');

-- --------------------------------------------------------

--
-- Table structure for table `document_requests`
--

CREATE TABLE `document_requests` (
  `id` int(11) NOT NULL,
  `student_id` int(11) NOT NULL,
  `document_type` varchar(100) NOT NULL,
  `purpose` text DEFAULT NULL,
  `copies` int(11) DEFAULT 1,
  `status` enum('pending','processing','ready','released','rejected') DEFAULT 'pending',
  `requested_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `processed_at` timestamp NULL DEFAULT NULL,
  `notes` text DEFAULT NULL,
  `document_file` varchar(255) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `document_requests`
--

INSERT INTO `document_requests` (`id`, `student_id`, `document_type`, `purpose`, `copies`, `status`, `requested_at`, `processed_at`, `notes`, `document_file`) VALUES
(18, 70, 'Transcript of Records', 'Transfer to Another School', 1, '', '2026-02-19 06:35:22', '2026-02-19 06:35:47', '', 'uploads\\documents\\transcript_of_records_13_1771482947.pdf');

-- --------------------------------------------------------

--
-- Table structure for table `enrollment_applications`
--

CREATE TABLE `enrollment_applications` (
  `id` int(11) NOT NULL,
  `student_id` int(11) NOT NULL,
  `academic_year` varchar(20) NOT NULL,
  `year_level` int(11) NOT NULL,
  `semester` varchar(10) NOT NULL,
  `course_id` int(11) NOT NULL,
  `subjects` text NOT NULL,
  `total_units` int(11) DEFAULT 0,
  `scholarship_status` varchar(50) DEFAULT 'non-scholar',
  `status` varchar(20) DEFAULT 'pending',
  `remarks` text DEFAULT NULL,
  `applied_at` datetime DEFAULT current_timestamp(),
  `processed_at` datetime DEFAULT NULL,
  `processed_by` int(11) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `enrollment_applications`
--

INSERT INTO `enrollment_applications` (`id`, `student_id`, `academic_year`, `year_level`, `semester`, `course_id`, `subjects`, `total_units`, `scholarship_status`, `status`, `remarks`, `applied_at`, `processed_at`, `processed_by`) VALUES
(1, 64, '2024-2025', 1, '2nd', 44, '86', 3, 'non-scholar', 'approved', NULL, '2026-02-18 11:54:49', '2026-02-18 14:56:00', NULL),
(2, 70, '2024-2025', 1, '2nd', 44, '86', 3, 'non-scholar', 'approved', NULL, '2026-02-18 15:56:31', '2026-02-18 15:57:25', NULL);

-- --------------------------------------------------------

--
-- Table structure for table `grades`
--

CREATE TABLE `grades` (
  `id` int(11) NOT NULL,
  `student_id` int(11) NOT NULL,
  `subject_id` int(11) NOT NULL,
  `teacher_id` int(11) NOT NULL,
  `prelim` decimal(5,2) DEFAULT NULL,
  `midterm` decimal(5,2) DEFAULT NULL,
  `finals` decimal(5,2) DEFAULT NULL,
  `remarks` text DEFAULT NULL,
  `is_released` tinyint(1) DEFAULT 0,
  `released_at` datetime DEFAULT NULL,
  `created_at` datetime DEFAULT current_timestamp(),
  `updated_at` datetime DEFAULT current_timestamp() ON UPDATE current_timestamp()
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `grades`
--

INSERT INTO `grades` (`id`, `student_id`, `subject_id`, `teacher_id`, `prelim`, `midterm`, `finals`, `remarks`, `is_released`, `released_at`, `created_at`, `updated_at`) VALUES
(4, 64, 84, 91, 3.00, 3.00, 3.00, '', 1, '2026-02-17 12:57:48', '2026-02-17 12:57:07', '2026-02-17 13:17:45'),
(5, 67, 84, 91, 3.00, 3.00, 3.00, '', 1, '2026-02-18 15:32:01', '2026-02-18 15:31:40', '2026-02-18 15:32:01'),
(6, 70, 84, 91, 3.00, 3.00, 3.00, '', 1, '2026-02-18 15:55:22', '2026-02-18 15:55:02', '2026-02-18 15:55:22');

-- --------------------------------------------------------

--
-- Table structure for table `lesson_materials`
--

CREATE TABLE `lesson_materials` (
  `id` int(11) NOT NULL,
  `teacher_id` int(11) NOT NULL,
  `subject_id` int(11) NOT NULL,
  `class_id` int(11) NOT NULL,
  `title` varchar(255) NOT NULL,
  `description` text DEFAULT NULL,
  `type` enum('video','image') NOT NULL,
  `file_name` varchar(255) NOT NULL,
  `file_path` varchar(500) NOT NULL,
  `file_size` bigint(20) NOT NULL,
  `due_date` datetime DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp()
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `lesson_materials`
--

INSERT INTO `lesson_materials` (`id`, `teacher_id`, `subject_id`, `class_id`, `title`, `description`, `type`, `file_name`, `file_path`, `file_size`, `due_date`, `created_at`, `updated_at`) VALUES
(9, 91, 77, 14, 'asd', 'asd', 'image', '91_1770905257_aa32cbdf49d5476e93d6704f19d9e011.jpeg', 'uploads\\lessons\\91_1770905257_aa32cbdf49d5476e93d6704f19d9e011.jpeg', 439692, '2026-03-02 04:22:00', '2026-02-12 14:07:37', '2026-02-12 14:07:37');

-- --------------------------------------------------------

--
-- Table structure for table `password_reset_tokens`
--

CREATE TABLE `password_reset_tokens` (
  `id` int(11) NOT NULL,
  `student_id` int(11) NOT NULL,
  `token` varchar(64) NOT NULL,
  `expires_at` datetime NOT NULL,
  `used` tinyint(1) DEFAULT 0,
  `created_at` datetime DEFAULT current_timestamp()
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `password_reset_tokens`
--

INSERT INTO `password_reset_tokens` (`id`, `student_id`, `token`, `expires_at`, `used`, `created_at`) VALUES
(15, 64, 'b8003a75775baf552ec2e4d4500e1d6415f7caf0561124dad4314fd61834925d', '2026-02-17 20:36:24', 1, '2026-02-17 19:36:24'),
(16, 64, 'c940d727b8e0286e73aa91208ad4b0025c7478faf35e83ea988024e83e30e2db', '2026-02-18 12:05:54', 1, '2026-02-18 11:05:54'),
(17, 64, '8410b34e48969230f007eae58badddee2b3b324735312aa7270019a4c33315c7', '2026-02-18 19:55:17', 0, '2026-02-18 18:55:17');

-- --------------------------------------------------------

--
-- Table structure for table `payment_fees`
--

CREATE TABLE `payment_fees` (
  `id` int(11) NOT NULL,
  `payment_id` int(11) NOT NULL,
  `fee_name` varchar(100) DEFAULT NULL,
  `amount` int(11) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `payment_fees`
--

INSERT INTO `payment_fees` (`id`, `payment_id`, `fee_name`, `amount`) VALUES
(211, 44, 'Miscellaneous Fee', 800),
(212, 44, 'Library Fee', 300),
(213, 44, 'Laboratory Fee', 1200),
(214, 44, 'Athletic Fee', 250),
(215, 44, 'Technology Fee', 600),
(216, 45, 'Miscellaneous Fee', 800),
(217, 45, 'Library Fee', 300),
(218, 45, 'Laboratory Fee', 1200),
(219, 45, 'Athletic Fee', 250),
(220, 45, 'Technology Fee', 600),
(221, 46, 'Miscellaneous Fee', 800),
(222, 46, 'Library Fee', 300),
(223, 46, 'Laboratory Fee', 1200),
(224, 46, 'Athletic Fee', 250),
(225, 46, 'Technology Fee', 600);

-- --------------------------------------------------------

--
-- Table structure for table `records_announcements`
--

CREATE TABLE `records_announcements` (
  `id` int(11) NOT NULL,
  `records_officer_id` int(11) NOT NULL,
  `title` varchar(255) NOT NULL,
  `content` text NOT NULL,
  `image_name` varchar(255) DEFAULT NULL,
  `image_path` varchar(500) DEFAULT NULL,
  `image_size` bigint(20) DEFAULT NULL,
  `priority` enum('low','normal','high','urgent') DEFAULT 'normal',
  `target_audience` enum('all','students','teachers') DEFAULT 'all',
  `is_active` tinyint(1) DEFAULT 1,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp()
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- Table structure for table `registrar_announcements`
--

CREATE TABLE `registrar_announcements` (
  `id` int(11) NOT NULL,
  `registrar_id` int(11) NOT NULL,
  `title` varchar(255) NOT NULL,
  `content` text NOT NULL,
  `target_audience` enum('all','pending','approved','enrolled') NOT NULL DEFAULT 'all',
  `image_name` varchar(255) DEFAULT NULL,
  `image_path` varchar(500) DEFAULT NULL,
  `image_size` bigint(20) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp()
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

--
-- Dumping data for table `registrar_announcements`
--

INSERT INTO `registrar_announcements` (`id`, `registrar_id`, `title`, `content`, `target_audience`, `image_name`, `image_path`, `image_size`, `created_at`, `updated_at`) VALUES
(6, 72, 'SCHEDULE OF MIDTERM EXAMINATIONS', 'ANNOUNCEMENT', 'all', 'announcement.jpg', 'uploads\\registrar-announcements\\72_1770967640_76e1c139079248ed8cee1a8590c7b326.jpg', 62243, '2026-02-13 07:27:20', '2026-02-13 07:27:20');

-- --------------------------------------------------------

--
-- Table structure for table `roles`
--

CREATE TABLE `roles` (
  `id` int(11) NOT NULL,
  `name` varchar(50) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `roles`
--

INSERT INTO `roles` (`id`, `name`) VALUES
(1, 'admin'),
(5, 'cashier'),
(6, 'guidance'),
(7, 'parent'),
(4, 'registrar'),
(2, 'student'),
(3, 'teacher');

-- --------------------------------------------------------

--
-- Table structure for table `school_year`
--

CREATE TABLE `school_year` (
  `id` int(11) NOT NULL,
  `year` varchar(20) NOT NULL,
  `semester` enum('1st','2nd','Summer') NOT NULL,
  `is_active` tinyint(1) DEFAULT 1
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- Table structure for table `sections`
--

CREATE TABLE `sections` (
  `id` int(11) NOT NULL,
  `section_id` int(11) NOT NULL,
  `section_name` varchar(50) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `sections`
--

INSERT INTO `sections` (`id`, `section_id`, `section_name`) VALUES
(14, 0, '23232'),
(15, 0, 'er'),
(16, 0, 'set A'),
(17, 0, 'bscs A'),
(18, 0, 'bscs As'),
(19, 0, 'asda'),
(20, 0, '1');

-- --------------------------------------------------------

--
-- Table structure for table `students`
--

CREATE TABLE `students` (
  `id` int(11) NOT NULL,
  `student_id` varchar(50) NOT NULL,
  `password` varchar(255) NOT NULL,
  `first_name` varchar(100) NOT NULL,
  `middle_name` varchar(100) DEFAULT NULL,
  `last_name` varchar(100) NOT NULL,
  `age` int(11) DEFAULT 18,
  `contact_number` varchar(50) DEFAULT NULL,
  `email` varchar(150) DEFAULT NULL,
  `address` text DEFAULT NULL,
  `status` enum('pending','approved','rejected') DEFAULT 'pending',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `profile_picture` varchar(255) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `students`
--

INSERT INTO `students` (`id`, `student_id`, `password`, `first_name`, `middle_name`, `last_name`, `age`, `contact_number`, `email`, `address`, `status`, `created_at`, `profile_picture`) VALUES
(64, '2200907', '$2a$10$cZTcz2iDOMGX6TQyuhz7uu2gz9dG0QeMQXh5evKmFpMgtlFZ2QMXm', 'Glen', 'F', 'Tagubase', 22, '09671139281', 'glen.tagubase03@gmail.com', '420 Advincula Ave.', 'approved', '2026-02-17 04:34:59', NULL),
(65, '21', '$2a$10$IlVkqkOtgc1lti.OOctgEOxFWcCX4fUgTdFV95COzTtOsy6UNZqKW', 'John', 'Patrick F.', 'Tagubase', 21, '09671139281', 'glen.tagubase03@gmail.com', '420 Advincula Ave.', 'approved', '2026-02-17 11:35:23', NULL),
(66, '321', '$2a$10$mismA7Z5AnVxRjYB9Dr9cOycPKj/sItyqfVIPkMvN4XTfmQ2r4gVW', 'John', 'Patrick F.', 'Tagubase', 21, '09671139281', 'glen.tagubase03@gmail.com', '420 Advincula Ave.', 'approved', '2026-02-18 07:08:30', NULL),
(67, '12345', '$2a$10$VNt8j3xF9D8AShfaSaquSejFogRgdYdTNsp3ymH/4fkB9M1hPKoAq', 'John', 'Patrick F.', 'Tagubase', 32, '09671139281', 'nakatagosabase@yahoo.com', '420 Advincula Ave.', 'approved', '2026-02-18 07:18:37', NULL),
(68, '111', '$2a$10$Pe6VP/w/1VAAUEBXKm7A7.ukH/vXvEcuVdL6xM81AQaLZB6gsGyMG', 'John', 'Patrick F.', 'Tagubase', 21, '09671139281', 'nakatagosabase@yahoo.com', '420 Advincula Ave.', 'approved', '2026-02-18 07:33:21', NULL),
(69, '12', '$2a$10$xrtTyzSim8Y2qUhepAGSP.2HNYp5m3NxUJoDQgipqqbksDCksLOt2', 'John', 'Patrick F.', 'Tagubase', 22, '09671139281', 'glen.tagubase03@gmail.com', '420 Advincula Ave.', 'approved', '2026-02-18 07:43:35', NULL),
(70, '13', '$2a$10$d3ldvdyvAhZHx0T4M4H6geYbi5BWP5Jvd6dyHyH6FGCzSMQdriVHy', 'John', 'Patrick F.', 'Tagubase', 12, '09671139281', 'nakatagosabase@yahoo.com', '420 Advincula Ave.', 'approved', '2026-02-18 07:50:58', NULL),
(72, '', '$2a$10$rWxTIgBszuzPoI3mmxKicOlZUD8MszjztiXF/8kYcPA4ciaHHCixW', '', '', '', 18, '', '', '', 'pending', '2026-02-20 13:03:41', NULL);

-- --------------------------------------------------------

--
-- Table structure for table `student_academic`
--

CREATE TABLE `student_academic` (
  `id` int(11) NOT NULL,
  `student_id` int(11) NOT NULL,
  `last_school_attended` varchar(255) DEFAULT NULL,
  `last_school_year` varchar(50) DEFAULT NULL,
  `course` varchar(100) DEFAULT NULL,
  `subjects` text DEFAULT NULL,
  `year_level` varchar(10) DEFAULT NULL,
  `semester` varchar(20) DEFAULT NULL,
  `scholarship_status` varchar(50) DEFAULT NULL,
  `total_units` int(11) DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `student_academic`
--

INSERT INTO `student_academic` (`id`, `student_id`, `last_school_attended`, `last_school_year`, `course`, `subjects`, `year_level`, `semester`, `scholarship_status`, `total_units`) VALUES
(60, 69, 'um', '2024', '44', '84', '1', '1st', 'non-scholar', 3),
(61, 70, 'um', '2024', '44', '86', '1', '2nd', 'non-scholar', 3),
(63, 72, '', '', '', '', '1', '1st', 'non-scholar', 0);

-- --------------------------------------------------------

--
-- Table structure for table `student_family`
--

CREATE TABLE `student_family` (
  `id` int(11) NOT NULL,
  `student_id` int(11) NOT NULL,
  `father_first_name` varchar(100) DEFAULT NULL,
  `father_middle_name` varchar(100) DEFAULT NULL,
  `father_last_name` varchar(100) DEFAULT NULL,
  `father_occupation` varchar(150) DEFAULT NULL,
  `father_contact_number` varchar(50) DEFAULT NULL,
  `father_address` text DEFAULT NULL,
  `mother_first_name` varchar(100) DEFAULT NULL,
  `mother_middle_name` varchar(100) DEFAULT NULL,
  `mother_last_name` varchar(100) DEFAULT NULL,
  `mother_occupation` varchar(150) DEFAULT NULL,
  `mother_contact_number` varchar(50) DEFAULT NULL,
  `mother_address` text DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `student_family`
--

INSERT INTO `student_family` (`id`, `student_id`, `father_first_name`, `father_middle_name`, `father_last_name`, `father_occupation`, `father_contact_number`, `father_address`, `mother_first_name`, `mother_middle_name`, `mother_last_name`, `mother_occupation`, `mother_contact_number`, `mother_address`) VALUES
(55, 64, 'John', 'Patrick F.', 'Tagubase', 'employee', '21313213', '420 Advincula Ave.', 'John', 'F', 'Tagubase', 'employee', '213131', '420 Advincula Ave.'),
(56, 65, 'John', 'Patrick F.', 'Tagubase', 'employee', '21313213', '420 Advincula Ave.', 'John', 'Patrick F.', 'Tagubase', 'a', '213131', '420 Advincula Ave.'),
(57, 66, 'John', 'Patrick F.', 'Tagubase', 'employee', '21313213', '420 Advincula Ave.', 'John', 'Patrick F.', 'Tagubase', 'employee', '213131', '420 Advincula Ave.'),
(58, 67, 'John', 'Patrick F.', 'Tagubase', 'employee', '21313213', '420 Advincula Ave.', 'John', 'Patrick F.', 'Tagubase', 'employee', '213131', '420 Advincula Ave.'),
(59, 68, 'John', 'Patrick F.', 'Tagubase', 'employee', '21313213', '420 Advincula Ave.', 'John', 'Patrick F.', 'Tagubase', 'employee', '213131', '420 Advincula Ave.'),
(60, 69, 'John', 'Patrick F.', 'Tagubase', 'employee', '21313213', '420 Advincula Ave.', 'John', 'Patrick F.', 'Tagubase', 'employee', '213131', '420 Advincula Ave.'),
(61, 70, 'John', 'Patrick F.', 'Tagubase', 'employee', '21313213', '420 Advincula Ave.', 'John', 'Patrick F.', 'Tagubase', 'employee', '213131', '420 Advincula Ave.'),
(63, 72, '', '', '', '', '', '', '', '', '', '', '', '');

-- --------------------------------------------------------

--
-- Table structure for table `student_installments`
--

CREATE TABLE `student_installments` (
  `id` int(11) NOT NULL,
  `payment_id` int(11) NOT NULL,
  `term` varchar(50) NOT NULL,
  `amount` decimal(10,2) NOT NULL,
  `status` varchar(20) DEFAULT 'pending',
  `paid_at` datetime DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp()
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `student_installments`
--

INSERT INTO `student_installments` (`id`, `payment_id`, `term`, `amount`, `status`, `paid_at`, `created_at`) VALUES
(73, 44, 'prelim', 850.00, 'paid', '2026-02-18 15:44:17', '2026-02-18 07:44:17'),
(74, 44, 'midterm', 850.00, 'pending', NULL, '2026-02-18 07:44:17'),
(75, 44, 'finals', 850.00, 'pending', NULL, '2026-02-18 07:44:17'),
(76, 45, 'prelim', 850.00, 'paid', '2026-02-18 15:52:40', '2026-02-18 07:51:45'),
(77, 45, 'midterm', 850.00, 'paid', '2026-02-18 15:53:22', '2026-02-18 07:51:45'),
(78, 45, 'finals', 850.00, 'pending', NULL, '2026-02-18 07:51:45');

-- --------------------------------------------------------

--
-- Table structure for table `student_payments`
--

CREATE TABLE `student_payments` (
  `id` int(11) NOT NULL,
  `student_id` int(11) NOT NULL,
  `semester` varchar(50) DEFAULT NULL,
  `school_year` varchar(50) DEFAULT NULL,
  `payment_method` varchar(50) DEFAULT NULL,
  `total_amount` int(11) NOT NULL,
  `amount_paid` int(11) DEFAULT 0,
  `status` varchar(20) DEFAULT 'unpaid',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `payment_type` enum('downpayment','full') DEFAULT 'downpayment',
  `downpayment_amount` decimal(10,2) NOT NULL DEFAULT 0.00
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `student_payments`
--

INSERT INTO `student_payments` (`id`, `student_id`, `semester`, `school_year`, `payment_method`, `total_amount`, `amount_paid`, `status`, `created_at`, `payment_type`, `downpayment_amount`) VALUES
(44, 69, '1st', '2025-2026', 'cash', 5550, 3000, 'partial', '2026-02-18 07:43:47', 'downpayment', 0.00),
(45, 70, '1st', '2025-2026', 'cash', 5550, 5550, 'paid', '2026-02-18 07:51:11', 'downpayment', 0.00),
(46, 70, '2nd', '2024-2025', NULL, 5550, 0, 'unpaid', '2026-02-18 07:57:25', 'downpayment', 0.00);

-- --------------------------------------------------------

--
-- Table structure for table `student_submissions`
--

CREATE TABLE `student_submissions` (
  `id` int(11) NOT NULL,
  `material_id` int(11) NOT NULL,
  `student_id` int(11) NOT NULL,
  `file_name` varchar(255) NOT NULL,
  `file_path` varchar(500) NOT NULL,
  `file_size` bigint(20) NOT NULL,
  `submitted_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `status` enum('on_time','late') DEFAULT 'on_time',
  `remarks` text DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- Table structure for table `subjects`
--

CREATE TABLE `subjects` (
  `id` int(11) NOT NULL,
  `subject_name` varchar(255) NOT NULL,
  `code` varchar(50) NOT NULL,
  `course_id` int(11) NOT NULL,
  `year_level` int(11) NOT NULL DEFAULT 1,
  `semester` varchar(10) DEFAULT NULL,
  `prerequisite_id` int(11) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `subjects`
--

INSERT INTO `subjects` (`id`, `subject_name`, `code`, `course_id`, `year_level`, `semester`, `prerequisite_id`) VALUES
(75, 'THS 103', 'THS 103', 44, 4, NULL, NULL),
(76, 'SE 102', 'SE 102', 44, 4, NULL, NULL),
(77, 'IAS 101', 'IAS 101', 44, 4, NULL, NULL),
(78, 'dzs', 'dzs', 44, 1, NULL, NULL),
(79, 'dz', 'dz', 44, 1, NULL, NULL),
(80, 'sda', 'sda', 44, 1, NULL, NULL),
(81, 'dsa', 'dsa', 44, 2, NULL, NULL),
(83, '3', '3', 44, 2, '2nd', NULL),
(84, '12', '12', 44, 1, '1st', NULL),
(85, 'Computer science cs102', 'Computer science cs102', 44, 2, '1st', NULL),
(86, 'sa', 'sa', 44, 1, '2nd', NULL);

-- --------------------------------------------------------

--
-- Table structure for table `submissions`
--

CREATE TABLE `submissions` (
  `id` int(11) NOT NULL,
  `assignment_id` int(11) DEFAULT NULL,
  `student_id` int(11) DEFAULT NULL,
  `file_path` varchar(255) DEFAULT NULL,
  `submitted_at` datetime DEFAULT NULL,
  `grade` float DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- Table structure for table `teacher_subjects`
--

CREATE TABLE `teacher_subjects` (
  `id` int(11) NOT NULL,
  `teacher_id` int(11) NOT NULL,
  `subject_id` int(11) NOT NULL,
  `course_id` int(11) NOT NULL,
  `room` varchar(50) DEFAULT NULL,
  `day` varchar(20) NOT NULL,
  `time_start` time NOT NULL,
  `time_end` time NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp()
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `teacher_subjects`
--

INSERT INTO `teacher_subjects` (`id`, `teacher_id`, `subject_id`, `course_id`, `room`, `day`, `time_start`, `time_end`, `created_at`, `updated_at`) VALUES
(19, 91, 84, 44, '32', 'Monday', '04:12:00', '14:41:00', '2026-02-17 04:37:14', '2026-02-17 04:37:14'),
(20, 91, 86, 44, '32', 'Tuesday', '02:14:00', '16:00:00', '2026-02-18 03:53:52', '2026-02-18 03:53:52');

-- --------------------------------------------------------

--
-- Table structure for table `users`
--

CREATE TABLE `users` (
  `id` int(11) NOT NULL,
  `username` varchar(50) DEFAULT NULL,
  `password` varchar(255) DEFAULT NULL,
  `first_name` varchar(255) NOT NULL,
  `middle_name` varchar(255) NOT NULL,
  `surname` varchar(255) NOT NULL,
  `email` varchar(255) NOT NULL,
  `contact_number` varchar(20) NOT NULL,
  `status` varchar(20) NOT NULL DEFAULT 'active',
  `role` varchar(50) NOT NULL DEFAULT 'admin',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp()
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

--
-- Dumping data for table `users`
--

INSERT INTO `users` (`id`, `username`, `password`, `first_name`, `middle_name`, `surname`, `email`, `contact_number`, `status`, `role`, `created_at`) VALUES
(1, 'admin', '$2a$14$mtsnBuCo0tA0RSvemhQaEewA2vBYqKAjk3nXBk67arspinPGvHCDS', '', '', '', '', '', 'active', 'admin', '2026-02-15 12:01:09'),
(3, 'teacher1', '$2a$14$TmDzGUzhEl8r0kC8Sw5KoeoU5.4ky4HRtnXMs4.6iqM18oLyHzfHG', '', '', '', '', '', 'active', 'admin', '2026-02-15 12:01:09'),
(4, 'registrar1', '$2a$14$FSSjh2Y4RlHwslQ9GLMHduShh0CbXo5YcC6ojZDqmNEYhhqcm.p4G', '', '', '', '', '', 'active', 'admin', '2026-02-15 12:01:09'),
(72, 'a', '$2a$14$Vq1hB1AZtlWDYL5Zm11wzOZwnbCop9SrA0U1r7w./ZCE3ZxiaXAXu', '', '', '', '', '', 'active', 'registrar', '2026-02-15 12:01:09'),
(87, 'casher', '$2a$14$y1F2zkWkoT7v9BChVXGscuoITceSfMvIPSwFTqe8EJ0c5Tc4KfZ.e', '', '', '', '', '', 'active', 'cashier', '2026-02-15 12:01:09'),
(91, 'PROF. ERVIN RAMOS', '$2a$14$JDGnunenGSOpW/RkobvwHu.fX88lbH9CVHD6.A3zZ08VIKrXxs8ZW', '', '', '', '', '', 'active', 'teacher', '2026-02-15 12:01:09'),
(93, 'PROF. ROGELIO PLAZA', '$2a$14$xhSgF7zTYs5fEw2VX0WY2u8SV9XePPTBskfLFgy.KSyO.QM3UDCde', '', '', '', '', '', 'active', 'teacher', '2026-02-15 12:01:09'),
(94, 'PROF. JOSELITO BORCES', '$2a$14$LDdwdwuK1OxF6GLEpUJ/ku1CM8.yYGDsNY8seu4bLMyRIuTqGxXta', '', '', '', '', '', 'active', 'teacher', '2026-02-15 12:01:09'),
(119, 'cashier1', '$2a$14$YMnfRr.2PXMewGou4JGIVOAuGNU6bUmgDTxBctMiUI6kfwzOXjbiy', '', '', '', '', '', 'active', 'cashier', '2026-02-15 12:01:09'),
(130, 'tess', '$2a$14$7ppft4L7IVGfeuX2dMrdGuoBaWinq4uknIf6kNdjAHATH7SmkAxbK', '', '', '', '', '', 'active', 'faculty', '2026-02-15 12:01:09'),
(131, 'records1', '$2a$14$VzKGyFw1AHgrFKl/YwAPKeEKQFiILYAIFjclaxFfkyln6HTou1EpS', '', '', '', '', '', 'active', 'records', '2026-02-15 12:01:09'),
(132, 'parent1', '$2a$14$GPKXfPrQY/ePqaqkrg9F8OTV9goMS6jGwIHc2fD9VTIG3Da8dbRDa', '', '', '', '', '', 'active', 'parent', '2026-02-15 12:01:09'),
(135, 'PROF TEST', '$2a$14$bvLP4jnajeb31oo6T1hxo.zzAN1GdCjoLvrNEwOjIc03gO5c.gaQW', 'hey', 'tes', 'tess', 'nakatagosabase@yahoo.com', '0491249124', 'active', 'teacher', '2026-02-15 13:32:18'),
(137, 'TEST1', '$2a$14$2YZMlOL7JnUGyIfdeJKrseZF1D8q7O2NPsGQZRLvZyRUsrwwuXHZa', 'tester', 'tester', 'testers', 'dsa@gmail.com', '3123213', 'active', 'records', '2026-02-15 13:53:23'),
(139, 'student1', '$2a$14$XVQO.25rKcmj8wLF2AO71.Th9Wi6.olCCbvru9xF7NyxvzEEBpgFu', '', '', '', '', '', 'active', 'student', '2026-02-18 13:02:05');

--
-- Indexes for dumped tables
--

--
-- Indexes for table `announcements`
--
ALTER TABLE `announcements`
  ADD PRIMARY KEY (`id`),
  ADD KEY `class_id` (`class_id`),
  ADD KEY `subject_id` (`subject_id`);

--
-- Indexes for table `class_schedules`
--
ALTER TABLE `class_schedules`
  ADD PRIMARY KEY (`id`),
  ADD KEY `subject_id` (`subject_id`);

--
-- Indexes for table `courses`
--
ALTER TABLE `courses`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `code` (`code`);

--
-- Indexes for table `document_requests`
--
ALTER TABLE `document_requests`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_document_requests_status` (`status`),
  ADD KEY `idx_document_requests_student` (`student_id`);

--
-- Indexes for table `enrollment_applications`
--
ALTER TABLE `enrollment_applications`
  ADD PRIMARY KEY (`id`),
  ADD KEY `course_id` (`course_id`),
  ADD KEY `idx_enrollment_student` (`student_id`),
  ADD KEY `idx_enrollment_status` (`status`);

--
-- Indexes for table `grades`
--
ALTER TABLE `grades`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `unique_grade` (`student_id`,`subject_id`,`teacher_id`),
  ADD KEY `subject_id` (`subject_id`),
  ADD KEY `teacher_id` (`teacher_id`);

--
-- Indexes for table `lesson_materials`
--
ALTER TABLE `lesson_materials`
  ADD PRIMARY KEY (`id`),
  ADD KEY `idx_class_id` (`class_id`),
  ADD KEY `idx_due_date` (`due_date`);

--
-- Indexes for table `password_reset_tokens`
--
ALTER TABLE `password_reset_tokens`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `token` (`token`),
  ADD KEY `idx_token` (`token`),
  ADD KEY `idx_student_id` (`student_id`);

--
-- Indexes for table `payment_fees`
--
ALTER TABLE `payment_fees`
  ADD PRIMARY KEY (`id`),
  ADD KEY `payment_id` (`payment_id`);

--
-- Indexes for table `records_announcements`
--
ALTER TABLE `records_announcements`
  ADD PRIMARY KEY (`id`),
  ADD KEY `records_officer_id` (`records_officer_id`);

--
-- Indexes for table `registrar_announcements`
--
ALTER TABLE `registrar_announcements`
  ADD PRIMARY KEY (`id`),
  ADD KEY `registrar_id` (`registrar_id`),
  ADD KEY `idx_target_audience` (`target_audience`),
  ADD KEY `idx_created_at` (`created_at`);

--
-- Indexes for table `roles`
--
ALTER TABLE `roles`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `name` (`name`);

--
-- Indexes for table `school_year`
--
ALTER TABLE `school_year`
  ADD PRIMARY KEY (`id`);

--
-- Indexes for table `sections`
--
ALTER TABLE `sections`
  ADD PRIMARY KEY (`id`);

--
-- Indexes for table `students`
--
ALTER TABLE `students`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `student_id` (`student_id`);

--
-- Indexes for table `student_academic`
--
ALTER TABLE `student_academic`
  ADD PRIMARY KEY (`id`),
  ADD KEY `student_id` (`student_id`);

--
-- Indexes for table `student_family`
--
ALTER TABLE `student_family`
  ADD PRIMARY KEY (`id`),
  ADD KEY `student_id` (`student_id`);

--
-- Indexes for table `student_installments`
--
ALTER TABLE `student_installments`
  ADD PRIMARY KEY (`id`),
  ADD KEY `payment_id` (`payment_id`);

--
-- Indexes for table `student_payments`
--
ALTER TABLE `student_payments`
  ADD PRIMARY KEY (`id`),
  ADD KEY `student_id` (`student_id`);

--
-- Indexes for table `student_submissions`
--
ALTER TABLE `student_submissions`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `unique_submission` (`material_id`,`student_id`),
  ADD KEY `idx_material_id` (`material_id`),
  ADD KEY `idx_student_id` (`student_id`);

--
-- Indexes for table `subjects`
--
ALTER TABLE `subjects`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `code` (`code`),
  ADD KEY `course_id` (`course_id`);

--
-- Indexes for table `submissions`
--
ALTER TABLE `submissions`
  ADD PRIMARY KEY (`id`),
  ADD KEY `fk_submissions_assignments` (`assignment_id`),
  ADD KEY `fk_submissions_enrollment` (`student_id`);

--
-- Indexes for table `teacher_subjects`
--
ALTER TABLE `teacher_subjects`
  ADD PRIMARY KEY (`id`),
  ADD KEY `fk_teacher` (`teacher_id`),
  ADD KEY `fk_subject` (`subject_id`),
  ADD KEY `fk_course` (`course_id`);

--
-- Indexes for table `users`
--
ALTER TABLE `users`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `username` (`username`);

--
-- AUTO_INCREMENT for dumped tables
--

--
-- AUTO_INCREMENT for table `announcements`
--
ALTER TABLE `announcements`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=4;

--
-- AUTO_INCREMENT for table `class_schedules`
--
ALTER TABLE `class_schedules`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `courses`
--
ALTER TABLE `courses`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=45;

--
-- AUTO_INCREMENT for table `document_requests`
--
ALTER TABLE `document_requests`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=19;

--
-- AUTO_INCREMENT for table `enrollment_applications`
--
ALTER TABLE `enrollment_applications`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=3;

--
-- AUTO_INCREMENT for table `grades`
--
ALTER TABLE `grades`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=7;

--
-- AUTO_INCREMENT for table `lesson_materials`
--
ALTER TABLE `lesson_materials`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=10;

--
-- AUTO_INCREMENT for table `password_reset_tokens`
--
ALTER TABLE `password_reset_tokens`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=18;

--
-- AUTO_INCREMENT for table `payment_fees`
--
ALTER TABLE `payment_fees`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=226;

--
-- AUTO_INCREMENT for table `records_announcements`
--
ALTER TABLE `records_announcements`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=4;

--
-- AUTO_INCREMENT for table `registrar_announcements`
--
ALTER TABLE `registrar_announcements`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=7;

--
-- AUTO_INCREMENT for table `roles`
--
ALTER TABLE `roles`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=8;

--
-- AUTO_INCREMENT for table `school_year`
--
ALTER TABLE `school_year`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `sections`
--
ALTER TABLE `sections`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=21;

--
-- AUTO_INCREMENT for table `students`
--
ALTER TABLE `students`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=73;

--
-- AUTO_INCREMENT for table `student_academic`
--
ALTER TABLE `student_academic`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=64;

--
-- AUTO_INCREMENT for table `student_family`
--
ALTER TABLE `student_family`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=64;

--
-- AUTO_INCREMENT for table `student_installments`
--
ALTER TABLE `student_installments`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=79;

--
-- AUTO_INCREMENT for table `student_payments`
--
ALTER TABLE `student_payments`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=47;

--
-- AUTO_INCREMENT for table `student_submissions`
--
ALTER TABLE `student_submissions`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=7;

--
-- AUTO_INCREMENT for table `subjects`
--
ALTER TABLE `subjects`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=87;

--
-- AUTO_INCREMENT for table `submissions`
--
ALTER TABLE `submissions`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `teacher_subjects`
--
ALTER TABLE `teacher_subjects`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=21;

--
-- AUTO_INCREMENT for table `users`
--
ALTER TABLE `users`
  MODIFY `id` int(11) NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=140;

--
-- Constraints for dumped tables
--

--
-- Constraints for table `announcements`
--
ALTER TABLE `announcements`
  ADD CONSTRAINT `announcements_ibfk_1` FOREIGN KEY (`class_id`) REFERENCES `teacher_subjects` (`id`) ON DELETE CASCADE,
  ADD CONSTRAINT `announcements_ibfk_2` FOREIGN KEY (`subject_id`) REFERENCES `subjects` (`id`) ON DELETE CASCADE;

--
-- Constraints for table `class_schedules`
--
ALTER TABLE `class_schedules`
  ADD CONSTRAINT `class_schedules_ibfk_1` FOREIGN KEY (`subject_id`) REFERENCES `subjects` (`id`) ON DELETE CASCADE;

--
-- Constraints for table `document_requests`
--
ALTER TABLE `document_requests`
  ADD CONSTRAINT `document_requests_ibfk_1` FOREIGN KEY (`student_id`) REFERENCES `students` (`id`);

--
-- Constraints for table `enrollment_applications`
--
ALTER TABLE `enrollment_applications`
  ADD CONSTRAINT `enrollment_applications_ibfk_1` FOREIGN KEY (`student_id`) REFERENCES `students` (`id`),
  ADD CONSTRAINT `enrollment_applications_ibfk_2` FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`);

--
-- Constraints for table `grades`
--
ALTER TABLE `grades`
  ADD CONSTRAINT `grades_ibfk_1` FOREIGN KEY (`student_id`) REFERENCES `students` (`id`) ON DELETE CASCADE,
  ADD CONSTRAINT `grades_ibfk_2` FOREIGN KEY (`subject_id`) REFERENCES `subjects` (`id`) ON DELETE CASCADE,
  ADD CONSTRAINT `grades_ibfk_3` FOREIGN KEY (`teacher_id`) REFERENCES `users` (`id`) ON DELETE CASCADE;

--
-- Constraints for table `password_reset_tokens`
--
ALTER TABLE `password_reset_tokens`
  ADD CONSTRAINT `password_reset_tokens_ibfk_1` FOREIGN KEY (`student_id`) REFERENCES `students` (`id`) ON DELETE CASCADE;

--
-- Constraints for table `payment_fees`
--
ALTER TABLE `payment_fees`
  ADD CONSTRAINT `payment_fees_ibfk_1` FOREIGN KEY (`payment_id`) REFERENCES `student_payments` (`id`);

--
-- Constraints for table `records_announcements`
--
ALTER TABLE `records_announcements`
  ADD CONSTRAINT `records_announcements_ibfk_1` FOREIGN KEY (`records_officer_id`) REFERENCES `users` (`id`) ON DELETE CASCADE;

--
-- Constraints for table `registrar_announcements`
--
ALTER TABLE `registrar_announcements`
  ADD CONSTRAINT `registrar_announcements_ibfk_1` FOREIGN KEY (`registrar_id`) REFERENCES `users` (`id`) ON DELETE CASCADE;

--
-- Constraints for table `student_academic`
--
ALTER TABLE `student_academic`
  ADD CONSTRAINT `student_academic_ibfk_1` FOREIGN KEY (`student_id`) REFERENCES `students` (`id`) ON DELETE CASCADE;

--
-- Constraints for table `student_family`
--
ALTER TABLE `student_family`
  ADD CONSTRAINT `student_family_ibfk_1` FOREIGN KEY (`student_id`) REFERENCES `students` (`id`) ON DELETE CASCADE;

--
-- Constraints for table `student_installments`
--
ALTER TABLE `student_installments`
  ADD CONSTRAINT `student_installments_ibfk_1` FOREIGN KEY (`payment_id`) REFERENCES `student_payments` (`id`);

--
-- Constraints for table `student_payments`
--
ALTER TABLE `student_payments`
  ADD CONSTRAINT `student_payments_ibfk_1` FOREIGN KEY (`student_id`) REFERENCES `students` (`id`);

--
-- Constraints for table `student_submissions`
--
ALTER TABLE `student_submissions`
  ADD CONSTRAINT `student_submissions_ibfk_1` FOREIGN KEY (`material_id`) REFERENCES `lesson_materials` (`id`) ON DELETE CASCADE,
  ADD CONSTRAINT `student_submissions_ibfk_2` FOREIGN KEY (`student_id`) REFERENCES `students` (`id`) ON DELETE CASCADE;
COMMIT;

/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
