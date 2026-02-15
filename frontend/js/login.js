const API = "";

// Display current date
function displayDate() {
    const options = { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' };
    const dateStr = new Date().toLocaleDateString('en-US', options);
    document.getElementById('currentDate').textContent = dateStr;
}

// Tab switching
function switchTab(tab) {
    const tabs = document.querySelectorAll('.nav-tab');
    const sections = document.querySelectorAll('.content-section');
    
    tabs.forEach(t => t.classList.remove('active'));
    sections.forEach(s => s.classList.remove('active'));
    
    if (tab === 'login') {
        tabs[0].classList.add('active');
        document.getElementById('loginSection').classList.add('active');
    } else {
        tabs[1].classList.add('active');
        document.getElementById('registerSection').classList.add('active');
    }
}

// ---------------- SUBJECT SELECTION ----------------
let subjectsData = [];

function updateSelectedSubjects() {
    const subjectsSelect = document.getElementById('subjects');
    const selectedOptions = Array.from(subjectsSelect.selectedOptions);
    const display = document.getElementById('selectedSubjectsDisplay');
    const list = document.getElementById('selectedSubjectsList');
    const count = document.getElementById('selectedCount');
    
    if (selectedOptions.length > 0) {
        display.classList.add('show');
        count.textContent = selectedOptions.length;
        
        list.innerHTML = '';
        selectedOptions.forEach(option => {
            const tag = document.createElement('span');
            tag.className = 'subject-tag';
            tag.innerHTML = `
                ${option.textContent}
                <span class="subject-tag-remove" onclick="removeSubject('${option.value}')">✕</span>
            `;
            list.appendChild(tag);
        });
    } else {
        display.classList.remove('show');
    }
    
    updateProgress();
}

function removeSubject(subjectId) {
    const subjectsSelect = document.getElementById('subjects');
    const option = Array.from(subjectsSelect.options).find(opt => opt.value === subjectId);
    if (option) {
        option.selected = false;
        updateSelectedSubjects();
    }
}

// ---------------- PROGRESS TRACKING ----------------
function updateProgress() {
    const requiredFields = [
        'student_id', 'password', 'first_name', 'last_name',
        'age', 'contact_number', 'email', 'address',
        'last_school_attended', 'last_school_year', 'course', 'year_level'
    ];
    
    let filled = 0;
    requiredFields.forEach(field => {
        const el = document.getElementById(field);
        if (el && el.value.trim()) filled++;
    });
    
    const selectedSubjects = Array.from(document.getElementById('subjects').selectedOptions);
    if (selectedSubjects.length > 0) filled++;
    
    const total = requiredFields.length + 1;
    const percent = Math.round((filled / total) * 100);
    
    const indicator = document.getElementById('progressIndicator');
    const fill = document.getElementById('progressFill');
    const label = document.getElementById('progressPercent');
    
    if (filled > 0) {
        indicator.style.display = 'block';
        fill.style.width = percent + '%';
        label.textContent = percent + '%';
    }
}

// ---------------- LOGIN ----------------
async function login() {
    const btn = event.target;
    const originalHTML = btn.innerHTML;
    const msgEl = document.getElementById('loginMsg');
    
    msgEl.className = 'message-box';
    msgEl.innerHTML = '';
    
    const loginId = document.getElementById('login_id').value;
    const password = document.getElementById('login_password').value;
    
    if (!loginId || !password) {
        msgEl.className = 'message-box error';
        msgEl.innerHTML = '⚠️ Please enter both student ID and password.';
        return;
    }
    
    btn.innerHTML = '<span class="spinner"></span> Authenticating...';
    btn.disabled = true;
    
    try {
        const res = await fetch(API + "/login", {
            method: "POST",
            credentials: "include",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ login_id: loginId, password })
        });
        
        const data = await res.json();
        
        if (res.ok) {
            msgEl.className = 'message-box success';
            msgEl.innerHTML = '✓ Login successful. Redirecting...';
            
            if (data.role === "student" && data.student_id) {
                localStorage.setItem("student_id", data.student_id);
            }
            
            setTimeout(() => window.location.href = data.redirect, 1200);
        } else {
            msgEl.className = 'message-box error';
            msgEl.innerHTML = '✗ ' + (data.error || 'Invalid credentials.');
        }
    } catch {
        msgEl.className = 'message-box error';
        msgEl.innerHTML = '⚠️ Server connection error.';
    } finally {
        btn.innerHTML = originalHTML;
        btn.disabled = false;
    }
}

// ---------------- LOAD COURSES ----------------
async function loadCourses() {
    try {
        const res = await fetch(API + "/admin/courses");
        const courses = await res.json();
        
        const select = document.getElementById('course');
        select.innerHTML = "<option value=''>Select a course</option>";
        
        courses.forEach(c => {
            const opt = document.createElement("option");
            opt.value = c.id;
            opt.textContent = c.course_name;
            select.appendChild(opt);
        });
    } catch (err) {
        console.error(err);
    }
}

// ---------------- LOAD SUBJECTS ----------------
async function loadSubjects() {
    const course = document.getElementById('course').value;
    const year = document.getElementById('year_level').value;
    const select = document.getElementById('subjects');
    
    if (!course || !year) {
        select.innerHTML = "<option disabled>Please select course and year level first</option>";
        updateSelectedSubjects();
        return;
    }
    
    try {
        const res = await fetch(`${API}/admin/subjects?course_id=${course}&year_level=${year}`);
        const subjects = await res.json();
        
        select.innerHTML = "";
        subjects.forEach(s => {
            const opt = document.createElement("option");
            opt.value = s.id;
            opt.textContent = s.subject_name;
            select.appendChild(opt);
        });
        
        updateSelectedSubjects();
    } catch {
        select.innerHTML = "<option disabled>Error loading subjects</option>";
    }
}

document.getElementById('course').addEventListener('change', loadSubjects);
document.getElementById('year_level').addEventListener('change', loadSubjects);

// ---------------- REGISTER ----------------
async function register() {
    const btn = event.target;
    const originalHTML = btn.innerHTML;
    const msgEl = document.getElementById('registerMsg');
    
    msgEl.className = 'message-box';
    msgEl.innerHTML = '';
    
    const selectedSubjects = Array.from(document.getElementById('subjects').selectedOptions).map(o => o.value);
    
    if (selectedSubjects.length === 0) {
        msgEl.className = 'message-box error';
        msgEl.innerHTML = '⚠️ Please select at least one subject.';
        return;
    }
    
    btn.innerHTML = '<span class="spinner"></span> Submitting...';
    btn.disabled = true;
    
    try {
        const res = await fetch(API + "/register-student", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ subjects: selectedSubjects })
        });
        
        const data = await res.json();
        
        if (res.ok) {
            msgEl.className = 'message-box success';
            msgEl.innerHTML = '✓ Registration submitted successfully.';
        } else {
            msgEl.className = 'message-box error';
            msgEl.innerHTML = data.error || 'Registration failed.';
        }
    } catch {
        msgEl.className = 'message-box error';
        msgEl.innerHTML = '⚠️ Connection error.';
    } finally {
        btn.innerHTML = originalHTML;
        btn.disabled = false;
    }
}

// Enter key login
document.getElementById('login_password').addEventListener('keypress', e => {
    if (e.key === 'Enter') login();
});

// Init
displayDate();
loadCourses();
