// ================= AUTH CHECK =================
const API = "http://localhost:8080";
const token = localStorage.getItem("token");
const role = localStorage.getItem("role");

if (!token || role !== "admin") {
  alert("Access denied. Please login.");
  window.location.href = "login.html";
}

// ================= USERS =================
async function showUsers() {
  document.getElementById("title").innerText = "Users";
  const output = document.getElementById("output");

  try {
    const res = await fetch(API + "/admin/users", {
      headers: { "Authorization": "Bearer " + token }
    });
    const users = await res.json();

    let html = `<table>
      <tr>
        <th>ID</th><th>Username</th><th>Role</th><th>Status</th><th>Actions</th>
      </tr>`;
    users.forEach(u => {
      html += `<tr>
        <td>${u.id}</td>
        <td>${u.username}</td>
        <td>${u.role}</td>
        <td>${u.status || "active"}</td>
        <td>
          <button onclick="editUserForm(${u.id}, '${u.username}', '${u.role}')">Edit</button>
          <button onclick="deleteUser(${u.id})">Delete</button>
        </td>
      </tr>`;
    });
    html += `</table>`;
    output.innerHTML = html;

  } catch (err) {
    output.innerHTML = "Error fetching users: " + err.message;
  }
}

function showCreateUser() {
  document.getElementById("title").innerText = "Create User";
  document.getElementById("output").innerHTML = `
    <input id="cu_username" placeholder="Username"><br><br>
    <input id="cu_password" type="password" placeholder="Password"><br><br>
    <select id="cu_role">
      <option value="student">Student</option>
      <option value="teacher">Teacher</option>
      <option value="registrar">Registrar</option>
      <option value="cashier">Cashier</option>
      <option value="guidance">Guidance</option>
      <option value="admin">Admin</option>
    </select><br><br>
    <button onclick="createUser()">Create</button>
    <p id="msg"></p>
  `;
}

async function createUser() {
  const username = document.getElementById("cu_username").value.trim();
  const password = document.getElementById("cu_password").value.trim();
  const role = document.getElementById("cu_role").value;
  const msg = document.getElementById("msg");

  if (!username || !password) {
    msg.innerText = "All fields required";
    msg.style.color = "red";
    return;
  }

  try {
    const res = await fetch(API + "/admin/users", {
      method: "POST",
      headers: { "Content-Type": "application/json", "Authorization": "Bearer " + token },
      body: JSON.stringify({ username, password, role })
    });
    const data = await res.json();
    if (!res.ok) {
      msg.innerText = data.error || "Failed to create user";
      msg.style.color = "red";
      return;
    }
    msg.innerText = "User created successfully";
    msg.style.color = "green";
    showUsers();
  } catch (err) {
    msg.innerText = "Server not reachable";
    msg.style.color = "red";
  }
}

function editUserForm(id, username, role) {
  document.getElementById("title").innerText = "Edit User";
  document.getElementById("output").innerHTML = `
    <input id="edit_username" value="${username}" placeholder="Username"><br><br>
    <input id="edit_password" type="password" placeholder="New Password (leave blank to keep)"><br><br>
    <select id="edit_role">
      <option value="student" ${role==='student'?'selected':''}>Student</option>
      <option value="teacher" ${role==='teacher'?'selected':''}>Teacher</option>
      <option value="registrar" ${role==='registrar'?'selected':''}>Registrar</option>
      <option value="cashier" ${role==='cashier'?'selected':''}>Cashier</option>
      <option value="guidance" ${role==='guidance'?'selected':''}>Guidance</option>
      <option value="admin" ${role==='admin'?'selected':''}>Admin</option>
    </select><br><br>
    <button onclick="editUser(${id})">Save Changes</button>
    <p id="msg"></p>
  `;
}

async function editUser(id) {
  const username = document.getElementById("edit_username").value.trim();
  const password = document.getElementById("edit_password").value.trim();
  const role = document.getElementById("edit_role").value;
  const msg = document.getElementById("msg");

  try {
    const res = await fetch(`${API}/admin/users/${id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json", "Authorization": "Bearer " + token },
      body: JSON.stringify({ username, password, role })
    });
    const data = await res.json();
    if (!res.ok) {
      msg.innerText = data.error || "Failed to edit user";
      msg.style.color = "red";
      return;
    }
    msg.innerText = "User updated successfully";
    msg.style.color = "green";
    showUsers();
  } catch (err) {
    msg.innerText = "Server not reachable";
    msg.style.color = "red";
  }
}

async function deleteUser(id) {
  if (!confirm("Delete this user?")) return;
  try {
    const res = await fetch(`${API}/admin/users/${id}`, {
      method: "DELETE",
      headers: { "Authorization": "Bearer " + token }
    });
    const data = await res.json();
    alert(data.message || data.error);
    showUsers();
  } catch (err) {
    alert("Server not reachable");
  }
}

// ================= LOGOUT =================
function logout() {
  localStorage.clear();
  window.location.href = "login.html";
}

// ================= PLACEHOLDER FUNCTIONS FOR OTHER SECTIONS =================
function showSubjects(){ document.getElementById("output").innerHTML="<h3>Subjects management here</h3>"; }
function showCourses(){ document.getElementById("output").innerHTML="<h3>Courses management here</h3>"; }
function showSections(){ document.getElementById("output").innerHTML="<h3>Sections management here</h3>"; }
function showAssignTeacher(){ document.getElementById("output").innerHTML="<h3>Assign teacher here</h3>"; }
function showSchoolYear(){ document.getElementById("output").innerHTML="<h3>Set school year here</h3>"; }

// ================= AUTO LOAD USERS ON START =================
showUsers();
