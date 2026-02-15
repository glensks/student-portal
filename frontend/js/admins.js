const API = "http://localhost:8080";

// ===== UTILS =====
function getToken() {
  const token = localStorage.getItem("jwt");
  if (!token) {
    alert("Login required");
    window.location.href = "/login.html";
    throw new Error("No token");
  }
  return token;
}

async function apiFetch(url, options = {}) {
  options.headers = {
    "Authorization": "Bearer " + getToken(),
    "Content-Type": "application/json"
  };
  const res = await fetch(API + url, options);
  let data = {};
  try { data = await res.json(); } catch {}
  if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`);
  return data;
}

function closeForm(id) {
  document.getElementById(id).style.display = "none";
}

function openForm(id) {
  document.getElementById(id).style.display = "flex";
}

// ===== USERS =====
document.getElementById("btnCreateUser").onclick = () => openForm("formCreateUser");

document.getElementById("submitCreateUser").onclick = async () => {
  const username = document.getElementById("userUsername").value.trim();
  const password = document.getElementById("userPassword").value.trim();
  const role = document.getElementById("userRole").value;
  if (!username || !password || !role) return alert("All fields required");

  try {
    const res = await apiFetch("/admin/users", {
      method: "POST",
      body: JSON.stringify({ username, password, role })
    });
    alert(res.message || "User created");
    closeForm("formCreateUser");
  } catch (e) { alert("Error: " + e.message); }
};

document.getElementById("btnViewUsers").onclick = async () => {
  openForm("formViewUsers");
  try {
    const users = await apiFetch("/admin/users");
    const ul = document.getElementById("userList");
    ul.innerHTML = "";
    users.forEach(u => ul.innerHTML += `<li>${u.username} - ${u.role} (${u.status})</li>`);
  } catch (e) { alert("Error: " + e.message); }
};

// ===== SUBJECTS =====
document.getElementById("btnCreateSubject").onclick = () => openForm("formCreateSubject");

document.getElementById("submitCreateSubject").onclick = async () => {
  const subject_name = document.getElementById("subjectName").value.trim();
  const code = document.getElementById("subjectCode").value.trim();
  if (!subject_name || !code) return alert("All fields required");

  try {
    const res = await apiFetch("/admin/subjects", {
      method: "POST",
      body: JSON.stringify({ subject_name, code })
    });
    alert(res.message || "Subject created");
    closeForm("formCreateSubject");
  } catch (e) { alert("Error: " + e.message); }
};

document.getElementById("btnViewSubjects").onclick = async () => {
  openForm("formViewSubjects");
  try {
    const subjects = await apiFetch("/admin/subjects");
    const ul = document.getElementById("subjectList");
    ul.innerHTML = "";
    subjects.forEach(s => ul.innerHTML += `<li>${s.subject_name} (${s.code})</li>`);
  } catch (e) { alert("Error: " + e.message); }
};

// ===== COURSES =====
document.getElementById("btnCreateCourse").onclick = () => openForm("formCreateCourse");

document.getElementById("submitCreateCourse").onclick = async () => {
  const course_name = document.getElementById("courseName").value.trim();
  const code = document.getElementById("courseCode").value.trim();
  if (!course_name || !code) return alert("All fields required");

  try {
    const res = await apiFetch("/admin/courses", {
      method: "POST",
      body: JSON.stringify({ course_name, code })
    });
    alert(res.message || "Course created");
    closeForm("formCreateCourse");
  } catch (e) { alert("Error: " + e.message); }
};

document.getElementById("btnViewCourses").onclick = async () => {
  openForm("formViewCourses");
  try {
    const courses = await apiFetch("/admin/courses");
    const ul = document.getElementById("courseList");
    ul.innerHTML = "";
    courses.forEach(c => ul.innerHTML += `<li>${c.course_name} (${c.code})</li>`);
  } catch (e) { alert("Error: " + e.message); }
};

// ===== SECTIONS =====
document.getElementById("btnCreateSection").onclick = () => openForm("formCreateSection");

document.getElementById("submitCreateSection").onclick = async () => {
  const section_name = document.getElementById("sectionNameInput").value.trim();
  if (!section_name) return alert("Section name required");

  try {
    const res = await apiFetch("/admin/sections", {
      method: "POST",
      body: JSON.stringify({ section_name })
    });
    alert(res.message || "Section created");
    closeForm("formCreateSection");
  } catch (e) { alert("Error: " + e.message); }
};

document.getElementById("btnViewSections").onclick = async () => {
  openForm("formViewSections");
  try {
    const sections = await apiFetch("/admin/sections");
    const ul = document.getElementById("sectionList");
    ul.innerHTML = "";
    sections.forEach(s => ul.innerHTML += `<li>${s.section_name}</li>`);
  } catch (e) { alert("Error: " + e.message); }
};

// ===== ASSIGN TEACHER =====
document.getElementById("btnAssignTeacher").onclick = async () => {
  openForm("formAssignTeacher");
  try {
    const teachers = await apiFetch("/admin/users?role=teacher");
    const subjects = await apiFetch("/admin/subjects");
    const sections = await apiFetch("/admin/sections");
    const courses = await apiFetch("/admin/courses");

    const t = document.getElementById("assignTeacher");
    t.innerHTML = "<option value=''>Select</option>";
    teachers.forEach(u => t.innerHTML += `<option value='${u.username}'>${u.username}</option>`);

    const s = document.getElementById("assignSubject");
    s.innerHTML = "<option value=''>Select</option>";
    subjects.forEach(sub => s.innerHTML += `<option value='${sub.subject_name}'>${sub.subject_name} (${sub.code})</option>`);

    const sec = document.getElementById("assignSection");
    sec.innerHTML = "<option value=''>Select</option>";
    sections.forEach(se => sec.innerHTML += `<option value='${se.section_name}'>${se.section_name}</option>`);

    const c = document.getElementById("assignCourse");
    c.innerHTML = "<option value=''>Select</option>";
    courses.forEach(co => c.innerHTML += `<option value='${co.course_name}'>${co.course_name} (${co.code})</option>`);

  } catch (e) { alert("Error: " + e.message); }
};

document.getElementById("submitAssignTeacher").onclick = async () => {
  const payload = {
    teacher_name: document.getElementById("assignTeacher").value,
    subject_name: document.getElementById("assignSubject").value,
    section_name: document.getElementById("assignSection").value,
    course_name: document.getElementById("assignCourse").value,
    day: document.getElementById("assignDay").value,
    time_start: document.getElementById("assignStart").value,
    time_end: document.getElementById("assignEnd").value
  };

  for (const key in payload) if (!payload[key]) return alert("All fields required");
  if (payload.time_start >= payload.time_end) return alert("End time must be later than start time");

  try {
    const res = await apiFetch("/admin/assign-teacher", {
      method: "POST",
      body: JSON.stringify(payload)
    });
    alert(res.message || "Teacher assigned");
    closeForm("formAssignTeacher");
  } catch (e) { alert("Error: " + e.message); }
};

// ===== SCHOOL YEAR =====
document.getElementById("btnSetSchoolYear").onclick = () => openForm("formSetSchoolYear");

document.getElementById("submitSchoolYear").onclick = async () => {
  const year = document.getElementById("schoolYear").value.trim();
  const semester = document.getElementById("schoolSemester").value;
  if (!year || !semester) return alert("All fields required");

  try {
    const res = await apiFetch("/admin/school-year", {
      method: "POST",
      body: JSON.stringify({ year, semester })
    });
    alert(res.message || "School year set");
    closeForm("formSetSchoolYear");
  } catch (e) { alert("Error: " + e.message); }
};
