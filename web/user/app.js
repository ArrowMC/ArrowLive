(() => {
  const LS_KEY = 'arrowlive.rooms';
  let currentRoom = null;
  let currentPlayer = null;
  let statusTimer = null;

  const $ = (id) => document.getElementById(id);

  function loadHistory() {
    try {
      const raw = localStorage.getItem(LS_KEY);
      return raw ? JSON.parse(raw) : [];
    } catch (_) {
      return [];
    }
  }

  function saveHistory(list) {
    localStorage.setItem(LS_KEY, JSON.stringify(list));
  }

  function upsertHistory(info) {
    const list = loadHistory().filter((r) => r.name !== info.name);
    list.unshift({ ...info, created_at: Date.now() });
    saveHistory(list.slice(0, 20));
    renderHistory();
  }

  function renderHistory() {
    const ul = $('historyList');
    const list = loadHistory();
    ul.innerHTML = '';
    if (list.length === 0) {
      ul.innerHTML = '<li class="small">暂无记录</li>';
      return;
    }
    for (const r of list) {
      const li = document.createElement('li');
      const when = new Date(r.created_at).toLocaleString();
      li.innerHTML = `
        <span class="name">${escapeHTML(r.name)}</span>
        <span class="small">${when}</span>
        <button class="copy" data-action="reopen">再次预览</button>
        <button class="copy" data-action="forget">删除</button>
      `;
      li.querySelector('[data-action="reopen"]').addEventListener('click', () => showRoom(r));
      li.querySelector('[data-action="forget"]').addEventListener('click', () => {
        const remain = loadHistory().filter((x) => x.name !== r.name);
        saveHistory(remain);
        renderHistory();
      });
      ul.appendChild(li);
    }
  }

  function escapeHTML(s) {
    return String(s).replace(/[&<>"']/g, (c) => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
  }

  async function createRoom() {
    const name = $('roomName').value.trim();
    $('createErr').textContent = '';
    if (!name) { $('createErr').textContent = '请输入房间名'; return; }
    try {
      const resp = await fetch('/user/api/rooms', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name }),
      });
      const data = await resp.json();
      if (!resp.ok) {
        $('createErr').textContent = data.error || '创建失败';
        return;
      }
      upsertHistory(data);
      showRoom(data);
    } catch (e) {
      $('createErr').textContent = String(e);
    }
  }

  function showRoom(info) {
    currentRoom = info;
    $('current').hidden = false;
    $('curName').textContent = info.name;
    $('curRtmp').textContent = info.rtmp_url;
    $('curToken').textContent = info.token;
    $('curPlay').textContent = info.play_url;
    setStatusBadge('waiting', '等待推流');
    startPlayer(info.play_url);
    startStatusPolling(info.name);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  function setStatusBadge(kind, text) {
    const el = $('curStatus');
    el.className = 'status ' + kind;
    el.textContent = text;
  }

  function startPlayer(url) {
    stopPlayer();
    const video = $('player');
    if (!window.mpegts || !mpegts.getFeatureList().mseLivePlayback) {
      video.src = url;
      return;
    }
    const p = mpegts.createPlayer(
      { type: 'mpegts', url, isLive: true },
      { enableStashBuffer: false, stashInitialSize: 128, liveBufferLatencyChasing: true }
    );
    p.attachMediaElement(video);
    p.load();
    p.play().catch(() => {});
    currentPlayer = p;
  }

  function stopPlayer() {
    if (currentPlayer) {
      try { currentPlayer.destroy(); } catch (_) {}
      currentPlayer = null;
    }
    const video = $('player');
    if (video) video.removeAttribute('src');
  }

  function startStatusPolling(name) {
    if (statusTimer) clearInterval(statusTimer);
    const poll = async () => {
      try {
        const r = await fetch(`/user/api/rooms/${encodeURIComponent(name)}/status`);
        if (!r.ok) return;
        const d = await r.json();
        if (d.status === 'active') setStatusBadge('live', '直播中');
        else if (d.status === 'inactive') setStatusBadge('offline', '已断开');
        else setStatusBadge('waiting', '等待推流');
      } catch (_) {}
    };
    poll();
    statusTimer = setInterval(poll, 3000);
  }

  function copyText(text, btn) {
    const flashOK = () => {
      const old = btn.textContent;
      btn.textContent = '已复制';
      setTimeout(() => (btn.textContent = old), 1200);
    };
    const flashErr = () => {
      const old = btn.textContent;
      btn.textContent = '复制失败';
      setTimeout(() => (btn.textContent = old), 1500);
    };

    // 现代 API：仅在安全上下文（https 或 localhost）可用
    if (navigator.clipboard && window.isSecureContext) {
      navigator.clipboard.writeText(text).then(flashOK).catch(() => legacyCopy(text, flashOK, flashErr));
      return;
    }
    legacyCopy(text, flashOK, flashErr);
  }

  function legacyCopy(text, ok, fail) {
    try {
      const ta = document.createElement('textarea');
      ta.value = text;
      ta.setAttribute('readonly', '');
      ta.style.position = 'fixed';
      ta.style.top = '-1000px';
      ta.style.opacity = '0';
      document.body.appendChild(ta);
      ta.select();
      ta.setSelectionRange(0, ta.value.length);
      const done = document.execCommand('copy');
      document.body.removeChild(ta);
      if (done) ok(); else fail();
    } catch (_) {
      fail();
    }
  }

  document.addEventListener('click', (e) => {
    const t = e.target;
    if (t.classList && t.classList.contains('copy') && t.dataset.target) {
      const text = document.getElementById(t.dataset.target).textContent;
      copyText(text, t);
    }
  });

  $('btnCreate').addEventListener('click', createRoom);
  $('roomName').addEventListener('keydown', (e) => { if (e.key === 'Enter') createRoom(); });
  renderHistory();
})();
