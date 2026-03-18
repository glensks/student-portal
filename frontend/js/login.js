/* ============================================================
   login.js — Student Information System Portal Logic
   ============================================================ */

const API = "";

/* ============================================================
   RATE LIMIT CONFIG (mirrors backend: 3 per 15 min)
   ============================================================ */
const FP_RL = {
    maxAttempts:    3,
    windowMs:       15 * 60 * 1000,
    blockMs:        15 * 60 * 1000,
    resendCooldown: 60,
    storageKey:     'fp_rl'
};

function fpRlLoad() {
    try { return JSON.parse(localStorage.getItem(FP_RL.storageKey)) || { attempts: [], blockedUntil: 0 }; }
    catch { return { attempts: [], blockedUntil: 0 }; }
}
function fpRlSave(d) { localStorage.setItem(FP_RL.storageKey, JSON.stringify(d)); }

function fpRlRecord() {
    const d = fpRlLoad(), now = Date.now();
    d.attempts = (d.attempts || []).filter(t => now - t < FP_RL.windowMs);
    d.attempts.push(now);
    if (d.attempts.length >= FP_RL.maxAttempts) d.blockedUntil = now + FP_RL.blockMs;
    fpRlSave(d);
}

function fpRlBlockedUntil() {
    const d = fpRlLoad(), now = Date.now();
    if (d.blockedUntil && now < d.blockedUntil) return d.blockedUntil;
    if (d.blockedUntil && now >= d.blockedUntil) fpRlSave({ attempts: [], blockedUntil: 0 });
    return 0;
}

function fpRlAttemptsLeft() {
    const d = fpRlLoad(), now = Date.now();
    const recent = (d.attempts || []).filter(t => now - t < FP_RL.windowMs);
    return Math.max(0, FP_RL.maxAttempts - recent.length);
}

/* ============================================================
   VALIDATION RULES
   ============================================================ */
const RULES = {
    name:        [/^[A-Za-zÀ-ÖØ-öø-ÿ\s'\-]*$/, 50,  'letters, spaces, hyphens, and apostrophes'],
    contact:     [/^[\d\s\+\-\(\)]*$/,          15,  'digits and phone characters only'],
    school_year: [/^[\d\-]*$/,                  9,   'format YYYY-YYYY (e.g. 2022-2023)'],
    text_gen:    [/^[A-Za-z0-9À-ÖØ-öø-ÿ\s',.\-\/\#\&\(\)]*$/, 200, 'letters, numbers, and common punctuation'],
    school_name: [/^[A-Za-zÀ-ÖØ-öø-ÿ0-9\s',.\-\(\)&]*$/, 100, 'letters, numbers, and basic punctuation'],
};

function setFieldError(inputEl, message) {
    inputEl.classList.remove('input-valid');
    inputEl.classList.add('input-error');
    let errEl = inputEl.parentElement.querySelector('.field-error');
    if (!errEl) errEl = inputEl.closest('.form-group') && inputEl.closest('.form-group').querySelector('.field-error');
    if (errEl) { errEl.textContent = '⚠ ' + message; errEl.classList.add('show'); }
}
function clearFieldError(inputEl) {
    inputEl.classList.remove('input-error');
    const group = inputEl.closest('.form-group');
    if (group) { const errEl = group.querySelector('.field-error'); if (errEl) errEl.classList.remove('show'); }
}
function setFieldValid(inputEl) { clearFieldError(inputEl); inputEl.classList.add('input-valid'); }

function attachFilter(id, ruleKey, extraValidation) {
    const el = document.getElementById(id);
    if (!el) return;
    const [regex, maxLen] = RULES[ruleKey];
    el.addEventListener('input', function () {
        const cursor   = this.selectionStart;
        const filtered = this.value.split('').filter(c => regex.test(c)).join('');
        this.value = filtered.length > maxLen ? filtered.slice(0, maxLen) : filtered;
        try { this.setSelectionRange(cursor, cursor); } catch(e) {}
        if (this.value.length === 0) { clearFieldError(this); this.classList.remove('input-valid'); }
        else if (extraValidation) { const msg = extraValidation(this.value); msg ? setFieldError(this, msg) : setFieldValid(this); }
        else { setFieldValid(this); }
        updateProgress();
    });
    el.addEventListener('blur', function () {
        if (this.value.length === 0) return;
        if (extraValidation) { const msg = extraValidation(this.value); msg ? setFieldError(this, msg) : setFieldValid(this); }
    });
}

function validateSchoolYear(val) { if (!/^\d{4}-\d{4}$/.test(val)) return 'Format must be YYYY-YYYY (e.g. 2022-2023).'; const [y1,y2]=val.split('-').map(Number); if(y2!==y1+1) return 'End year must be start year + 1.'; return null; }
function validateContact(val)    { const digits=val.replace(/\D/g,''); if(digits.length<7||digits.length>15) return 'Enter a valid phone number (7–15 digits).'; return null; }
function validateEmail(val)      { if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(val)) return 'Enter a valid email address.'; return null; }

/* ============================================================
   BIRTHDAY → AGE AUTO-COMPUTE
   ============================================================ */
function computeAgeFromBirthday() {
    const birthdayEl  = document.getElementById('birthday');
    const ageDisplay  = document.getElementById('age_display');
    const birthdayErr = document.getElementById('birthdayErr');
    const val         = birthdayEl.value;

    if (!val) {
        ageDisplay.value = '';
        birthdayEl.classList.remove('input-valid', 'input-error');
        if (birthdayErr) { birthdayErr.textContent = ''; birthdayErr.classList.remove('show'); }
        return;
    }

    const dob = new Date(val);
    const now = new Date();

    if (dob >= now) {
        ageDisplay.value = '';
        birthdayEl.classList.add('input-error');
        birthdayEl.classList.remove('input-valid');
        if (birthdayErr) { birthdayErr.textContent = '⚠ Date of birth cannot be in the future.'; birthdayErr.classList.add('show'); }
        return;
    }

    let age = now.getFullYear() - dob.getFullYear();
    const m = now.getMonth() - dob.getMonth();
    if (m < 0 || (m === 0 && now.getDate() < dob.getDate())) age--;

    if (age < 15 || age > 100) {
        ageDisplay.value = age + ' years old';
        birthdayEl.classList.add('input-error');
        birthdayEl.classList.remove('input-valid');
        if (birthdayErr) { birthdayErr.textContent = '⚠ Age must be between 15 and 100.'; birthdayErr.classList.add('show'); }
        return;
    }

    ageDisplay.value = age + ' years old';
    birthdayEl.classList.add('input-valid');
    birthdayEl.classList.remove('input-error');
    if (birthdayErr) { birthdayErr.textContent = ''; birthdayErr.classList.remove('show'); }
}

// Helper: get computed age as integer from birthday field (returns null if invalid)
function getComputedAge() {
    const val = document.getElementById('birthday').value;
    if (!val) return null;
    const dob = new Date(val);
    const now = new Date();
    if (dob >= now) return null;
    let age = now.getFullYear() - dob.getFullYear();
    const m = now.getMonth() - dob.getMonth();
    if (m < 0 || (m === 0 && now.getDate() < dob.getDate())) age--;
    return age;
}

/* ============================================================
   DOM READY
   ============================================================ */
document.addEventListener('DOMContentLoaded', function () {
    const loginId = document.getElementById('login_id');
    if (loginId) {
        loginId.addEventListener('input', function () {
            const filtered = this.value.replace(/[^A-Za-z0-9\-_.]/g, '');
            this.value = filtered.length > 50 ? filtered.slice(0, 50) : filtered;
            clearFieldError(this); this.classList.remove('input-valid');
        });
        loginId.addEventListener('blur', function () { if (this.value) setFieldValid(this); });
    }

    attachFilter('first_name',      'name',        v => v.trim().length < 2 ? 'First name must be at least 2 characters.' : null);
    attachFilter('middle_name',     'name',        null);
    attachFilter('last_name',       'name',        v => v.trim().length < 2 ? 'Last name must be at least 2 characters.' : null);
    attachFilter('contact_number',  'contact',     validateContact);

    const emailEl = document.getElementById('email');
    if (emailEl) {
        emailEl.addEventListener('input', function () {
            if (this.value.length > 100) this.value = this.value.slice(0, 100);
            if (this.value) { const msg = validateEmail(this.value); msg ? setFieldError(this, msg) : setFieldValid(this); }
            else { clearFieldError(this); this.classList.remove('input-valid'); }
            updateProgress();
        });
    }

    attachFilter('address',               'text_gen',    v => v.trim().length < 5 ? 'Please enter a complete address.' : null);
    attachFilter('father_first_name',     'name',        null);
    attachFilter('father_middle_name',    'name',        null);
    attachFilter('father_last_name',      'name',        null);
    attachFilter('father_occupation',     'text_gen',    null);
    attachFilter('father_contact_number', 'contact',     validateContact);
    const fAddr = document.getElementById('father_address');
    if (fAddr) fAddr.addEventListener('input', function () { const [regex,maxLen]=RULES.text_gen; const filtered=this.value.split('').filter(c=>regex.test(c)||c==='\n').join(''); this.value=filtered.length>maxLen?filtered.slice(0,maxLen):filtered; });

    attachFilter('mother_first_name',     'name',        null);
    attachFilter('mother_middle_name',    'name',        null);
    attachFilter('mother_last_name',      'name',        null);
    attachFilter('mother_occupation',     'text_gen',    null);
    attachFilter('mother_contact_number', 'contact',     validateContact);
    const mAddr = document.getElementById('mother_address');
    if (mAddr) mAddr.addEventListener('input', function () { const [regex,maxLen]=RULES.text_gen; const filtered=this.value.split('').filter(c=>regex.test(c)||c==='\n').join(''); this.value=filtered.length>maxLen?filtered.slice(0,maxLen):filtered; });

    attachFilter('last_school_attended', 'school_name', v => v.trim().length < 3 ? 'Please enter the school name.' : null);
    attachFilter('last_school_year',     'school_year',  validateSchoolYear);

    const forgotEmailEl = document.getElementById('forgot_email');
    if (forgotEmailEl) forgotEmailEl.addEventListener('keypress', e => { if (e.key === 'Enter') submitForgotPassword(); });

    const loginPw = document.getElementById('login_password');
    if (loginPw) loginPw.addEventListener('keypress', e => { if (e.key === 'Enter') login(); });

    const courseEl = document.getElementById('course');
    const yearEl   = document.getElementById('year_level');
    const semEl    = document.getElementById('semester');
    if (courseEl) courseEl.addEventListener('change', loadSubjects);
    if (yearEl)   yearEl.addEventListener('change',   loadSubjects);
    if (semEl)    semEl.addEventListener('change',    loadSubjects);

    // Prevent picking future dates
    const birthdayEl = document.getElementById('birthday');
    if (birthdayEl) birthdayEl.setAttribute('max', new Date().toISOString().split('T')[0]);

    initPage();
});

/* ============================================================
   DATE
   ============================================================ */
function displayDate() {
    document.getElementById('currentDate').textContent =
        new Date().toLocaleDateString('en-US', { weekday:'long', year:'numeric', month:'long', day:'numeric' });
}

/* ============================================================
   TABS
   ============================================================ */
function switchTab(tab) {
    document.querySelectorAll('.nav-tab').forEach(t => t.classList.remove('active'));
    document.querySelectorAll('.content-section').forEach(s => s.classList.remove('active'));
    if (tab === 'login') {
        document.querySelectorAll('.nav-tab')[0].classList.add('active');
        document.getElementById('loginSection').classList.add('active');
    } else {
        document.querySelectorAll('.nav-tab')[1].classList.add('active');
        document.getElementById('registerSection').classList.add('active');
    }
}

/* ============================================================
   PAGE INIT
   ============================================================ */
function initPage() {
    displayDate();
    loadCourses();
    const params = new URLSearchParams(window.location.search);
    const token  = params.get('token');
    if (token) {
        document.getElementById('portalContainer').style.display = 'none';
        document.getElementById('resetPage').classList.add('visible');
        verifyToken(token);
    }
}

/* ============================================================
   FORGOT PASSWORD MODAL
   ============================================================ */
let fpResendTimer = null;

function openForgotModal() {
    document.getElementById('forgotModal').classList.add('open');
    document.getElementById('forgotMsg').className = 'message-box';
    document.getElementById('forgotMsg').innerHTML = '';
    document.getElementById('forgot_email').value  = '';
    updateAttemptsHint();
    const blockedUntil = fpRlBlockedUntil();
    if (blockedUntil) { showModalStep(3); startModalBlockTimer(blockedUntil); }
    else { showModalStep(1); setTimeout(() => document.getElementById('forgot_email').focus(), 300); }
}

function closeForgotModal() {
    document.getElementById('forgotModal').classList.remove('open');
    clearInterval(fpResendTimer);
    setTimeout(() => showModalStep(1), 300);
}

function closeForgotModalOnOverlay(e) {
    if (e.target === document.getElementById('forgotModal')) closeForgotModal();
}

function showModalStep(step) {
    document.querySelectorAll('.modal-step').forEach(s => s.classList.remove('active'));
    const el = document.getElementById('modalStep' + step);
    if (el) el.classList.add('active');
}

document.addEventListener('keydown', e => { if (e.key === 'Escape') closeForgotModal(); });

function updateAttemptsHint() {
    const hint = document.getElementById('fpAttemptsHint');
    if (!hint) return;
    const left = fpRlAttemptsLeft();
    if (left === 1) { hint.textContent = '⚠️ 1 attempt remaining in this window.'; hint.style.color = '#d97706'; }
    else { hint.textContent = ''; }
}

/* ============================================================
   SUBMIT FORGOT PASSWORD
   FIX: Added 404 email_not_found handling — shows error message
        without counting as a rate-limit attempt.
        Only fpRlRecord() after a confirmed successful submission.
   ============================================================ */
async function submitForgotPassword() {
    const btn   = document.getElementById('forgotBtn');
    const msgEl = document.getElementById('forgotMsg');
    const email = document.getElementById('forgot_email').value.trim().toLowerCase();

    msgEl.className = 'message-box';
    msgEl.innerHTML = '';

    // 1. Client-side block check
    const blockedUntil = fpRlBlockedUntil();
    if (blockedUntil) { showModalStep(3); startModalBlockTimer(blockedUntil); return; }

    // 2. Empty check
    if (!email) {
        msgEl.className = 'message-box error';
        msgEl.innerHTML = '⚠️ Please enter your email address.';
        return;
    }

    // 3. Format check — validateEmail returns an error string or null
    if (validateEmail(email)) {
        msgEl.className = 'message-box error';
        msgEl.innerHTML = '⚠️ Please enter a valid email address.';
        return;
    }

    // 4. Loading state
    const original = btn.innerHTML;
    btn.innerHTML = '<span class="spinner"></span> Sending...';
    btn.disabled  = true;

    try {
        const res  = await fetch(API + '/forgot-password', {
            method:  'POST',
            headers: { 'Content-Type': 'application/json' },
            body:    JSON.stringify({ email })
        });
        const data = await res.json();

        // 5. Server-side rate limit — sync client block state
        if (res.status === 429) {
            const until = data.retry_after_seconds
                ? Date.now() + data.retry_after_seconds * 1000
                : Date.now() + FP_RL.blockMs;
            const d = fpRlLoad();
            d.blockedUntil = until;
            fpRlSave(d);
            showModalStep(3);
            startModalBlockTimer(until);
            return;
        }

        // 6. Email not registered — show error, do NOT count as an attempt
        //    so users can correct a typo without burning their 3 attempts.
        if (res.status === 404 && data.error === 'email_not_found') {
            msgEl.className = 'message-box error';
            msgEl.innerHTML = '⚠️ No account found with that email address. Please check and try again.';
            document.getElementById('forgot_email').focus();
            updateAttemptsHint();
            return;
        }

        // 7. Success — email was found and reset link was sent
        fpRlRecord();
        updateAttemptsHint();
        document.getElementById('sentToEmail').textContent   = email;
        document.getElementById('modalSubtitle').textContent = 'Reset email sent!';
        showModalStep(2);
        startResendCountdown();

    } catch (_) {
        msgEl.className = 'message-box error';
        msgEl.innerHTML = '⚠️ Unable to connect. Please check your connection.';
    } finally {
        btn.innerHTML = original;
        btn.disabled  = false;
    }
}

/* ============================================================
   RESEND COUNTDOWN
   ============================================================ */
function startResendCountdown() {
    const resendBtn = document.getElementById('resendBtn');
    const cdEl      = document.getElementById('resendCountdown');
    if (!resendBtn || !cdEl) return;
    let secs = FP_RL.resendCooldown;
    resendBtn.disabled = true;
    cdEl.textContent   = secs;
    clearInterval(fpResendTimer);
    fpResendTimer = setInterval(() => {
        secs--;
        cdEl.textContent = secs;
        if (secs <= 0) {
            clearInterval(fpResendTimer);
            resendBtn.disabled = false;
            resendBtn.innerHTML = 'Resend email';
        }
    }, 1000);
}

/* ============================================================
   RESEND RESET EMAIL
   FIX: Added proper status checks — only fpRlRecord() on success,
        handles 429 block sync, ignores 404 (email already verified
        at this point but guard kept for safety).
   ============================================================ */
async function resendReset() {
    const email = document.getElementById('sentToEmail').textContent;

    // Re-check block before resend
    const blockedUntil = fpRlBlockedUntil();
    if (blockedUntil) { showModalStep(3); startModalBlockTimer(blockedUntil); return; }

    const btn = document.getElementById('resendBtn');
    btn.disabled = true;
    btn.innerHTML = 'Sending...';

    try {
        const res  = await fetch(API + '/forgot-password', {
            method:  'POST',
            headers: { 'Content-Type': 'application/json' },
            body:    JSON.stringify({ email })
        });
        const data = await res.json();

        // Server-side rate limit hit during resend
        if (res.status === 429) {
            const until = data.retry_after_seconds
                ? Date.now() + data.retry_after_seconds * 1000
                : Date.now() + FP_RL.blockMs;
            const d = fpRlLoad();
            d.blockedUntil = until;
            fpRlSave(d);
            showModalStep(3);
            startModalBlockTimer(until);
            return;
        }

        // Only count the attempt when the server actually processed it
        if (res.ok) {
            fpRlRecord();
            updateAttemptsHint();
        }

    } catch (_) {
        // Network error — still restart the countdown so the button re-enables
    }

    startResendCountdown();
}

/* ============================================================
   BLOCKED STATE — live MM:SS countdown
   ============================================================ */
let fpBlockTimer = null;

function startModalBlockTimer(blockedUntil) {
    const el = document.getElementById('fpBlockedTimer');
    if (!el) return;
    clearInterval(fpBlockTimer);
    function tick() {
        const ms   = Math.max(0, blockedUntil - Date.now());
        const mins = Math.floor(ms / 60000);
        const secs = Math.floor((ms % 60000) / 1000);
        el.textContent = `${String(mins).padStart(2,'0')}:${String(secs).padStart(2,'0')}`;
        if (ms <= 0) {
            clearInterval(fpBlockTimer);
            showModalStep(1);
            document.getElementById('forgot_email').value = '';
            const hint = document.getElementById('fpAttemptsHint');
            if (hint) hint.textContent = '';
        }
    }
    tick();
    fpBlockTimer = setInterval(tick, 1000);
}

/* ============================================================
   RESET PASSWORD
   ============================================================ */
async function verifyToken(token) {
    try {
        const res  = await fetch(API + '/verify-reset-token?token=' + encodeURIComponent(token));
        const data = await res.json();
        if (!data.valid) {
            document.getElementById('resetFormArea').style.display = 'none';
            document.getElementById('invalidToken').style.display = 'block';
        }
    } catch (_) {}
}

function checkPasswordStrength() {
    const pw        = document.getElementById('new_password').value;
    const indicator = document.getElementById('strengthIndicator');
    const fill      = document.getElementById('strengthFill');
    const label     = document.getElementById('strengthLabel');
    if (pw.length === 0) { indicator.style.display = 'none'; return; }
    indicator.style.display = 'block';
    let score = 0;
    if (pw.length >= 8)           score++;
    if (pw.length >= 12)          score++;
    if (/[A-Z]/.test(pw))         score++;
    if (/[0-9]/.test(pw))         score++;
    if (/[^A-Za-z0-9]/.test(pw))  score++;
    const configs = [
        { width:'20%',  color:'#ef4444', text:'Very Weak' },
        { width:'40%',  color:'#f97316', text:'Weak' },
        { width:'60%',  color:'#eab308', text:'Fair' },
        { width:'80%',  color:'#22c55e', text:'Strong' },
        { width:'100%', color:'#16a34a', text:'Very Strong' },
    ];
    const cfg = configs[Math.min(score - 1, 4)] || configs[0];
    fill.style.width      = cfg.width;
    fill.style.background = cfg.color;
    label.textContent     = cfg.text;
    label.style.color     = cfg.color;
}

function checkPasswordMatch() {
    const pw      = document.getElementById('new_password').value;
    const confirm = document.getElementById('confirm_password').value;
    const msg     = document.getElementById('matchMsg');
    if (confirm.length === 0) { msg.innerHTML = ''; return; }
    msg.innerHTML = pw === confirm
        ? '<span style="color:#16a34a;">✓ Passwords match</span>'
        : '<span style="color:#dc2626;">✗ Passwords do not match</span>';
}

async function submitResetPassword() {
    const btn             = document.getElementById('resetBtn');
    const msgEl           = document.getElementById('resetMsg');
    const token           = new URLSearchParams(window.location.search).get('token');
    const newPassword     = document.getElementById('new_password').value;
    const confirmPassword = document.getElementById('confirm_password').value;

    msgEl.className = 'message-box';
    msgEl.innerHTML = '';

    if (!newPassword || newPassword.length < 8) {
        msgEl.className = 'message-box error';
        msgEl.innerHTML = '⚠️ Password must be at least 8 characters.';
        return;
    }
    if (newPassword !== confirmPassword) {
        msgEl.className = 'message-box error';
        msgEl.innerHTML = '⚠️ Passwords do not match.';
        return;
    }

    const original = btn.innerHTML;
    btn.innerHTML = '<span class="spinner"></span> Resetting...';
    btn.disabled  = true;

    try {
        const res  = await fetch(API + '/reset-password', {
            method:  'POST',
            headers: { 'Content-Type': 'application/json' },
            body:    JSON.stringify({ token, new_password: newPassword })
        });
        const data = await res.json();
        if (res.ok) {
            document.getElementById('resetFormArea').style.display = 'none';
            document.getElementById('resetSuccess').style.display  = 'block';
        } else {
            msgEl.className = 'message-box error';
            msgEl.innerHTML = '✗ ' + (data.error || 'Reset failed. The link may have expired.');
        }
    } catch (_) {
        msgEl.className = 'message-box error';
        msgEl.innerHTML = '⚠️ Unable to connect. Please try again.';
    } finally {
        btn.innerHTML = original;
        btn.disabled  = false;
    }
}

/* ============================================================
   LOGIN
   ============================================================ */
async function login() {
    const btn          = event.target;
    const originalHTML = btn.innerHTML;
    const msgEl        = document.getElementById('loginMsg');

    msgEl.className = 'message-box';
    msgEl.innerHTML = '';

    const loginId  = document.getElementById('login_id').value.trim();
    const password = document.getElementById('login_password').value;

    if (!loginId)  { setFieldError(document.getElementById('login_id'), 'Please enter your Student ID or Username.'); return; }
    if (!password) { msgEl.className = 'message-box error'; msgEl.innerHTML = '⚠️ Please enter your password.'; return; }

    btn.innerHTML = '<span class="spinner"></span> Authenticating...';
    btn.disabled  = true;

    try {
        const res  = await fetch(API + '/login', {
            method:      'POST',
            credentials: 'include',
            headers:     { 'Content-Type': 'application/json' },
            body:        JSON.stringify({ login_id: loginId, password })
        });
        const data = await res.json();
        if (res.ok) {
            localStorage.setItem('jwt', data.token);
            if (data.role === 'student' && data.student_id) localStorage.setItem('student_id', data.student_id);
            msgEl.className = 'message-box success';
            msgEl.innerHTML = '✓ Login successful. Redirecting...';
            setTimeout(() => { window.location.href = data.redirect; }, 1200);
        } else {
            msgEl.className = 'message-box error';
            msgEl.innerHTML = '✗ ' + (data.error || 'Invalid credentials.');
        }
    } catch (_) {
        msgEl.className = 'message-box error';
        msgEl.innerHTML = '⚠️ Unable to connect to the server.';
    } finally {
        btn.innerHTML = originalHTML;
        btn.disabled  = false;
    }
}

/* ============================================================
   REGISTER
   ============================================================ */
let subjectsData = [];

function updateSelectedSubjects() {
    const sel      = document.getElementById('subjects');
    const selected = Array.from(sel.selectedOptions);
    const display  = document.getElementById('selectedSubjectsDisplay');
    const list     = document.getElementById('selectedSubjectsList');
    const count    = document.getElementById('selectedCount');
    if (selected.length > 0) {
        display.classList.add('show');
        count.textContent = selected.length;
        list.innerHTML    = '';
        selected.forEach(opt => {
            const tag = document.createElement('span');
            tag.className = 'subject-tag';
            tag.innerHTML = `${opt.textContent}<span class="subject-tag-remove" onclick="removeSubject('${opt.value}')">✕</span>`;
            list.appendChild(tag);
        });
    } else {
        display.classList.remove('show');
    }
    updateProgress();
}

function removeSubject(id) {
    const opt = Array.from(document.getElementById('subjects').options).find(o => o.value === id);
    if (opt) { opt.selected = false; updateSelectedSubjects(); }
}

function updateProgress() {
    const fields = ['first_name','last_name','birthday','contact_number','email','address','last_school_attended','last_school_year','course','year_level','semester','scholarship_status'];
    let filled = fields.filter(f => { const el = document.getElementById(f); return el && el.value.trim(); }).length;
    const subjectsEl = document.getElementById('subjects');
    if (subjectsEl && Array.from(subjectsEl.selectedOptions).length > 0) filled++;
    const total     = fields.length + 1;
    const pct       = Math.round((filled / total) * 100);
    const indicator = document.getElementById('progressIndicator');
    if (indicator && filled > 0) {
        indicator.style.display = 'block';
        document.getElementById('progressFill').style.width    = pct + '%';
        document.getElementById('progressPercent').textContent = pct + '%';
    }
}

async function loadCourses() {
    try {
        const res  = await fetch(API + '/public/courses');
        if (!res.ok) { console.error('Failed to load courses, status:', res.status); return; }
        const data    = await res.json();
        const courses = Array.isArray(data) ? data : [];
        const sel     = document.getElementById('course');
        sel.innerHTML = "<option value=''>Select a course</option>";
        courses.forEach(c => {
            const opt = document.createElement('option');
            opt.value       = c.id;
            opt.textContent = c.course_name;
            sel.appendChild(opt);
        });
    } catch (err) { console.error('Error loading courses:', err); }
}

async function loadSubjects() {
    const course    = document.getElementById('course').value;
    const yearLevel = document.getElementById('year_level').value;
    const semester  = document.getElementById('semester').value;
    const prompt    = document.getElementById('subjectsPrompt');
    const sel       = document.getElementById('subjects');

    if (!course || !yearLevel || !semester) {
        prompt.style.display = 'block';
        prompt.innerHTML     = '<div class="prompt-icon">📚</div><div>Please select your <strong>Course</strong>, <strong>Year Level</strong>, and <strong>Semester</strong> first to load available subjects.</div>';
        sel.style.display    = 'none';
        sel.innerHTML        = '';
        document.getElementById('selectedSubjectsDisplay').classList.remove('show');
        return;
    }

    prompt.style.display = 'block';
    prompt.innerHTML     = '<div class="subjects-loading">⏳ Loading subjects...</div>';
    sel.style.display    = 'none';

    try {
        const res         = await fetch(`${API}/public/subjects?course_id=${course}&year_level=${yearLevel}&semester=${encodeURIComponent(semester)}`);
        const contentType = res.headers.get('content-type') || '';
        if (!res.ok || !contentType.includes('application/json')) throw new Error(`Server returned ${res.status}`);
        const subjects = await res.json();
        subjectsData   = subjects;

        prompt.style.display = 'none';
        sel.style.display    = 'block';
        sel.innerHTML        = '';

        if (subjects.length === 0) {
            prompt.style.display = 'block';
            prompt.innerHTML     = `<div class="prompt-icon">😕</div><div>No subjects found for <strong>${semester} Semester</strong> of your selected course and year level.</div>`;
            sel.style.display    = 'none';
        } else {
            subjects.forEach(s => {
                const opt       = document.createElement('option');
                opt.value       = s.id;
                opt.textContent = `${s.code ? s.code + ' — ' : ''}${s.subject_name || s.name || ''}`;
                sel.appendChild(opt);
            });
        }
        document.getElementById('selectedSubjectsDisplay').classList.remove('show');
        updateProgress();
    } catch (err) {
        prompt.style.display = 'block';
        prompt.innerHTML     = '<div class="prompt-icon">⚠️</div><div>Error loading subjects. Please try again.</div>';
        sel.style.display    = 'none';
        console.error(err);
    }
}

function validateRegisterForm(fields, age) {
    const errors = [];
    if (!fields.first_name || fields.first_name.trim().length < 2) errors.push('First name is required (min 2 characters).');
    if (!fields.last_name  || fields.last_name.trim().length < 2)  errors.push('Last name is required (min 2 characters).');
    if (!fields.birthday) {
        errors.push('Date of birth is required.');
    } else if (age === null || age < 15 || age > 100) {
        errors.push('Please enter a valid date of birth (age must be 15–100).');
    }
    if (!fields.contact_number) errors.push('Contact number is required.');
    else if (validateContact(fields.contact_number)) errors.push(validateContact(fields.contact_number));
    if (!fields.email) errors.push('Email is required.');
    else if (validateEmail(fields.email)) errors.push(validateEmail(fields.email));
    if (!fields.address || fields.address.trim().length < 5) errors.push('Complete address is required.');
    if (!fields.last_school_attended || fields.last_school_attended.trim().length < 3) errors.push('Last school attended is required.');
    if (!fields.last_school_year) errors.push('School year is required.');
    else if (validateSchoolYear(fields.last_school_year)) errors.push(validateSchoolYear(fields.last_school_year));
    if (!fields.course)             errors.push('Please select a course.');
    if (!fields.year_level)         errors.push('Please select a year level.');
    if (!fields.semester)           errors.push('Please select a semester.');
    if (!fields.scholarship_status) errors.push('Please select scholarship status.');
    return errors;
}

async function register() {
    const btn          = event.target;
    const originalHTML = btn.innerHTML;
    const msgEl        = document.getElementById('registerMsg');

    msgEl.className = 'message-box';
    msgEl.innerHTML = '';

    const fields = {
        first_name:           document.getElementById('first_name').value.trim(),
        middle_name:          document.getElementById('middle_name').value.trim(),
        last_name:            document.getElementById('last_name').value.trim(),
        birthday:             document.getElementById('birthday').value,
        contact_number:       document.getElementById('contact_number').value.trim(),
        email:                document.getElementById('email').value.trim(),
        address:              document.getElementById('address').value.trim(),
        last_school_attended: document.getElementById('last_school_attended').value.trim(),
        last_school_year:     document.getElementById('last_school_year').value.trim(),
        course:               document.getElementById('course').value,
        year_level:           document.getElementById('year_level').value,
        semester:             document.getElementById('semester').value,
        scholarship_status:   document.getElementById('scholarship_status').value,
    };

    const age    = getComputedAge();
    const errors = validateRegisterForm(fields, age);
    if (errors.length > 0) {
        msgEl.className = 'message-box error';
        msgEl.innerHTML = '⚠️ ' + errors[0];
        msgEl.scrollIntoView({ behavior:'smooth', block:'center' });
        return;
    }

    const selectedSubjects = Array.from(document.getElementById('subjects').selectedOptions).map(o => o.value);
    if (selectedSubjects.length === 0) {
        msgEl.className = 'message-box error';
        msgEl.innerHTML = '⚠️ Please select at least one subject.';
        msgEl.scrollIntoView({ behavior:'smooth', block:'center' });
        return;
    }

    const payload = {
        ...fields,
        age,
        father_first_name:     document.getElementById('father_first_name').value.trim(),
        father_middle_name:    document.getElementById('father_middle_name').value.trim(),
        father_last_name:      document.getElementById('father_last_name').value.trim(),
        father_occupation:     document.getElementById('father_occupation').value.trim(),
        father_contact_number: document.getElementById('father_contact_number').value.trim(),
        father_address:        document.getElementById('father_address').value.trim(),
        mother_first_name:     document.getElementById('mother_first_name').value.trim(),
        mother_middle_name:    document.getElementById('mother_middle_name').value.trim(),
        mother_last_name:      document.getElementById('mother_last_name').value.trim(),
        mother_occupation:     document.getElementById('mother_occupation').value.trim(),
        mother_contact_number: document.getElementById('mother_contact_number').value.trim(),
        mother_address:        document.getElementById('mother_address').value.trim(),
        subjects:              selectedSubjects,
    };

    // birthday is only used for frontend UX — not sent to backend
    delete payload.birthday;

    btn.innerHTML = '<span class="spinner"></span> Submitting Application...';
    btn.disabled  = true;

    try {
        const res  = await fetch(API + '/register-student', {
            method:  'POST',
            headers: { 'Content-Type': 'application/json' },
            body:    JSON.stringify(payload)
        });
        const data = await res.json();
        if (res.ok) {
            msgEl.className = 'message-box success';
            msgEl.innerHTML = '✓ Application submitted successfully! Your registration is now pending review.';
            msgEl.scrollIntoView({ behavior:'smooth', block:'center' });
            setTimeout(() => {
                document.querySelectorAll('.form-control').forEach(el => { el.value=''; el.classList.remove('input-valid','input-error'); });
                document.querySelectorAll('.field-error').forEach(el => el.classList.remove('show'));
                document.getElementById('progressIndicator').style.display = 'none';
                document.getElementById('selectedSubjectsDisplay').classList.remove('show');
                document.getElementById('subjects').style.display  = 'none';
                document.getElementById('subjectsPrompt').style.display = 'block';
                document.getElementById('subjectsPrompt').innerHTML = '<div class="prompt-icon">📚</div><div>Please select your <strong>Course</strong>, <strong>Year Level</strong>, and <strong>Semester</strong> first to load available subjects.</div>';
            }, 3000);
        } else {
            msgEl.className = 'message-box error';
            msgEl.innerHTML = '✗ ' + (data.error || 'Registration failed. Please try again.');
            msgEl.scrollIntoView({ behavior:'smooth', block:'center' });
        }
    } catch (_) {
        msgEl.className = 'message-box error';
        msgEl.innerHTML = '⚠️ Unable to submit application. Please check your connection.';
        msgEl.scrollIntoView({ behavior:'smooth', block:'center' });
    } finally {
        btn.innerHTML = originalHTML;
        btn.disabled  = false;
    }
}

/* ============================================================
   PASSWORD TOGGLE
   ============================================================ */
function togglePw(id, btn) {
    const el   = document.getElementById(id);
    const show = el.type === 'password';
    el.type    = show ? 'text' : 'password';
    btn.querySelector('.eye-show').style.display = show ? 'none' : '';
    btn.querySelector('.eye-hide').style.display = show ? ''     : 'none';
    btn.style.color = show ? 'var(--forest-700)' : '';
}