/* ============================================================
   forgot_password.js — University Student Portal
   ============================================================ */

const API = "";

/* ============================================================
   CLIENT-SIDE RATE LIMIT CONFIG
   mirrors backend: 3 attempts per 15-minute window
   ============================================================ */
const RL = {
    maxAttempts:    3,
    windowMs:       15 * 60 * 1000,  // 15 min
    blockMs:        15 * 60 * 1000,  // 15 min block
    resendCooldown: 60,              // seconds
    storageKey:     'fp_rl'
};

/* ============================================================
   RATE LIMIT HELPERS
   ============================================================ */
function rlLoad() {
    try { return JSON.parse(localStorage.getItem(RL.storageKey)) || { attempts: [], blockedUntil: 0 }; }
    catch { return { attempts: [], blockedUntil: 0 }; }
}

function rlSave(d) {
    localStorage.setItem(RL.storageKey, JSON.stringify(d));
}

function rlRecord() {
    const d = rlLoad(), now = Date.now();
    d.attempts = (d.attempts || []).filter(t => now - t < RL.windowMs);
    d.attempts.push(now);
    if (d.attempts.length >= RL.maxAttempts) {
        d.blockedUntil = now + RL.blockMs;
    }
    rlSave(d);
}

// Returns blockedUntil timestamp if still blocked, else 0
function rlBlockedUntil() {
    const d = rlLoad(), now = Date.now();
    if (d.blockedUntil && now < d.blockedUntil) return d.blockedUntil;
    if (d.blockedUntil && now >= d.blockedUntil) {
        rlSave({ attempts: [], blockedUntil: 0 }); // clear expired
    }
    return 0;
}

function rlAttemptsLeft() {
    const d = rlLoad(), now = Date.now();
    const recent = (d.attempts || []).filter(t => now - t < RL.windowMs);
    return Math.max(0, RL.maxAttempts - recent.length);
}

/* ============================================================
   STATE HELPERS
   ============================================================ */
function showState(id) {
    document.querySelectorAll('.state').forEach(s => s.classList.remove('active'));
    document.getElementById(id).classList.add('active');
}

function showMsg(type, html) {
    const el = document.getElementById('formMsg');
    el.className = 'msg-box ' + type + ' show';
    el.innerHTML = html;
}

function clearMsg() {
    const el = document.getElementById('formMsg');
    el.className = 'msg-box';
    el.innerHTML = '';
}

/* ============================================================
   EMAIL VALIDATION
   ============================================================ */
function isValidEmail(email) {
    return /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/.test(email);
}

function onEmailInput() {
    clearMsg();
    const left = rlAttemptsLeft();
    const hint = document.getElementById('attemptsHint');
    if (left === 1) {
        hint.textContent = '⚠️ 1 attempt remaining in this window.';
        hint.className = 'attempts-hint warn';
    } else {
        hint.textContent = '';
        hint.className = 'attempts-hint';
    }
}

/* ============================================================
   SUBMIT
   ============================================================ */
async function submitForgot() {
    const btn   = document.getElementById('submitBtn');
    const email = document.getElementById('emailInput').value.trim().toLowerCase();

    clearMsg();

    // 1. Client-side block check
    const blockedUntil = rlBlockedUntil();
    if (blockedUntil) { showBlockedState(blockedUntil); return; }

    // 2. Empty check
    if (!email) {
        showMsg('error', '⚠️ Please enter your email address.');
        document.getElementById('emailInput').focus();
        return;
    }

    // 3. Format validation
    if (!isValidEmail(email)) {
        showMsg('error', '⚠️ Please enter a valid email address (e.g. juan@example.com).');
        document.getElementById('emailInput').focus();
        return;
    }

    // 4. Loading state
    const orig = btn.innerHTML;
    btn.innerHTML = '<span class="spinner"></span> Sending...';
    btn.disabled  = true;
    document.getElementById('emailInput').disabled = true;

    try {
        const res  = await fetch(API + '/forgot-password', {
            method:  'POST',
            headers: { 'Content-Type': 'application/json' },
            body:    JSON.stringify({ email })
        });
        const data = await res.json();

        if (res.status === 429) {
            // Server-side rate limit hit — sync client
            const until = data.retry_after_seconds
                ? Date.now() + data.retry_after_seconds * 1000
                : Date.now() + RL.blockMs;
            const d = rlLoad();
            d.blockedUntil = until;
            rlSave(d);
            showBlockedState(until);
            return;
        }

        // Record attempt on any non-429 response
        rlRecord();

        // Always show sent screen — never reveal if email exists or not
        showSentState(email);

    } catch (_) {
        showMsg('error', '⚠️ Unable to connect. Please check your internet connection.');
    } finally {
        btn.innerHTML = orig;
        btn.disabled  = false;
        document.getElementById('emailInput').disabled = false;
    }
}

/* ============================================================
   SENT STATE + RESEND COUNTDOWN
   ============================================================ */
let resendTimer = null;

function showSentState(email) {
    document.getElementById('sentEmailDisplay').textContent = email;
    showState('stateSent');
    startResendCountdown();
}

function startResendCountdown() {
    const btn  = document.getElementById('resendBtn');
    const cdEl = document.getElementById('resendCountdown');
    let secs   = RL.resendCooldown;

    btn.disabled = true;
    cdEl.textContent = secs;

    clearInterval(resendTimer);
    resendTimer = setInterval(() => {
        secs--;
        cdEl.textContent = secs;
        if (secs <= 0) {
            clearInterval(resendTimer);
            btn.disabled = false;
            btn.innerHTML = 'Resend email';
        }
    }, 1000);
}

async function resendEmail() {
    const email = document.getElementById('sentEmailDisplay').textContent;

    // Re-check block before resend
    const blockedUntil = rlBlockedUntil();
    if (blockedUntil) { showBlockedState(blockedUntil); return; }

    const btn = document.getElementById('resendBtn');
    btn.disabled  = true;
    btn.innerHTML = 'Sending...';

    try {
        await fetch(API + '/forgot-password', {
            method:  'POST',
            headers: { 'Content-Type': 'application/json' },
            body:    JSON.stringify({ email })
        });
        rlRecord();
    } catch (_) {}

    startResendCountdown();
}

/* ============================================================
   BLOCKED STATE — live MM:SS countdown
   ============================================================ */
let blockTimer = null;

function showBlockedState(blockedUntil) {
    showState('stateBlocked');
    tickBlockTimer(blockedUntil);
}

function tickBlockTimer(blockedUntil) {
    const el = document.getElementById('blockedTimer');
    clearInterval(blockTimer);

    function tick() {
        const ms   = Math.max(0, blockedUntil - Date.now());
        const mins = Math.floor(ms / 60000);
        const secs = Math.floor((ms % 60000) / 1000);
        el.textContent = `${String(mins).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
        if (ms <= 0) {
            clearInterval(blockTimer);
            showState('stateForm');
            document.getElementById('emailInput').value = '';
            document.getElementById('attemptsHint').textContent = '';
            clearMsg();
        }
    }
    tick();
    blockTimer = setInterval(tick, 1000);
}

/* ============================================================
   INIT
   ============================================================ */
document.addEventListener('DOMContentLoaded', function () {
    // Date display
    document.getElementById('currentDate').textContent =
        new Date().toLocaleDateString('en-US', {
            weekday: 'long', year: 'numeric', month: 'long', day: 'numeric'
        });

    // Check if already blocked on page load
    const bu = rlBlockedUntil();
    if (bu) showBlockedState(bu);
});