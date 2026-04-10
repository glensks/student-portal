const API = "";

document.addEventListener('DOMContentLoaded', function () {
    document.getElementById('currentDate').textContent =
        new Date().toLocaleDateString('en-US', { weekday:'long', year:'numeric', month:'long', day:'numeric' });
    verifyToken();
});

function showState(id) {
    document.querySelectorAll('.state').forEach(s => s.classList.remove('active'));
    document.getElementById(id).classList.add('active');
}

function showMsg(type, html) {
    const el = document.getElementById('formMsg');
    el.className = 'msg-box ' + type + ' show';
    el.innerHTML = html;
}

async function verifyToken() {
    const token = new URLSearchParams(window.location.search).get('token');
    if (!token) { showState('stateInvalid'); return; }
    try {
        const res  = await fetch(API + '/verify-reset-token?token=' + encodeURIComponent(token));
        const data = await res.json();
        if (!data.valid) showState('stateInvalid');
    } catch (_) {}
}

function checkStrength() {
    const pw   = document.getElementById('new_password').value;
    const bar  = document.getElementById('strengthBar');
    const fill = document.getElementById('strengthFill');
    const lbl  = document.getElementById('strengthLabel');
    if (!pw) { bar.style.display = 'none'; return; }
    bar.style.display = 'block';
    let score = 0;
    if (pw.length >= 8)          score++;
    if (pw.length >= 12)         score++;
    if (/[A-Z]/.test(pw))        score++;
    if (/[0-9]/.test(pw))        score++;
    if (/[^A-Za-z0-9]/.test(pw)) score++;
    const cfg = [
        { w:'20%', c:'#ef4444', t:'Very Weak' },
        { w:'40%', c:'#f97316', t:'Weak' },
        { w:'60%', c:'#eab308', t:'Fair' },
        { w:'80%', c:'#22c55e', t:'Strong' },
        { w:'100%',c:'#16a34a', t:'Very Strong' },
    ][Math.min(score - 1, 4)] || { w:'20%', c:'#ef4444', t:'Very Weak' };
    fill.style.width      = cfg.w;
    fill.style.background = cfg.c;
    lbl.textContent       = cfg.t;
    lbl.style.color       = cfg.c;
}

function checkMatch() {
    const pw  = document.getElementById('new_password').value;
    const cfm = document.getElementById('confirm_password').value;
    const el  = document.getElementById('matchMsg');
    if (!cfm) { el.innerHTML = ''; return; }
    el.innerHTML = pw === cfm
        ? '<span style="color:#16a34a;">✓ Passwords match</span>'
        : '<span style="color:#dc2626;">✗ Passwords do not match</span>';
}

function togglePw(id) {
    const el = document.getElementById(id);
    el.type  = el.type === 'password' ? 'text' : 'password';
}

async function submitReset() {
    const btn   = document.getElementById('resetBtn');
    const token = new URLSearchParams(window.location.search).get('token');
    const pw    = document.getElementById('new_password').value;
    const cfm   = document.getElementById('confirm_password').value;

    if (!pw || pw.length < 8) { showMsg('error', '⚠️ Password must be at least 8 characters.'); return; }
    if (pw !== cfm)            { showMsg('error', '⚠️ Passwords do not match.'); return; }

    const orig    = btn.innerHTML;
    btn.innerHTML = '<span class="spinner"></span> Resetting...';
    btn.disabled  = true;

    try {
        const res  = await fetch(API + '/reset-password', {
            method:  'POST',
            headers: { 'Content-Type': 'application/json' },
            body:    JSON.stringify({ token, new_password: pw })
        });
        const data = await res.json();
        if (res.ok) { showState('stateSuccess'); }
        else        { showMsg('error', '✗ ' + (data.error || 'Reset failed. The link may have expired.')); }
    } catch (_) {
        showMsg('error', '⚠️ Unable to connect. Please try again.');
    } finally {
        btn.innerHTML = orig;
        btn.disabled  = false;
    }
}