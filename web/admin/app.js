(() => {
  const $ = (id) => document.getElementById(id);
  let mode = 'list';           // 'list' | 'grid'
  let rooms = [];              // {name, play_url}[]
  let groups = [];             // [][]
  let listPicked = null;       // room name
  let groupIdx = 0;
  let pollTimer = null;

  const listPlayers = []; // slot 0 for list mode
  const gridPlayers = []; // slots 0..3 for grid mode

  async function fetchJSON(url) {
    const r = await fetch(url, { credentials: 'same-origin' });
    if (r.status === 401) {
      location.href = '/admin/login.html';
      throw new Error('unauthorized');
    }
    if (!r.ok) throw new Error(await r.text());
    return r.json();
  }

  function destroyPlayer(slot, pool) {
    const p = pool[slot];
    if (p) {
      try { p.destroy(); } catch (_) {}
      pool[slot] = null;
    }
  }

  function attachPlayer(video, url, slot, pool) {
    destroyPlayer(slot, pool);
    if (!url) return;
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
    pool[slot] = p;
  }

  function renderList() {
    const ul = $('roomList');
    ul.innerHTML = '';
    $('roomEmpty').hidden = rooms.length > 0;
    for (const r of rooms) {
      const li = document.createElement('li');
      li.textContent = r.name;
      if (r.name === listPicked) li.classList.add('active');
      li.addEventListener('click', () => {
        listPicked = r.name;
        renderList();
        playListRoom(r);
      });
      ul.appendChild(li);
    }
    if (!listPicked && rooms.length > 0) {
      listPicked = rooms[0].name;
      renderList();
      playListRoom(rooms[0]);
    } else if (listPicked && !rooms.find((x) => x.name === listPicked)) {
      // previously picked room gone
      listPicked = rooms[0] ? rooms[0].name : null;
      renderList();
      if (listPicked) playListRoom(rooms[0]);
      else {
        destroyPlayer(0, listPlayers);
        $('listPlayer').removeAttribute('src');
        $('listPlayerLabel').textContent = '';
      }
    }
  }

  function playListRoom(r) {
    $('listPlayerLabel').textContent = '正在播放：' + r.name;
    attachPlayer($('listPlayer'), r.play_url, 0, listPlayers);
  }

  function renderGroupTabs() {
    const box = $('groupTabs');
    box.innerHTML = '';
    if (groups.length === 0) {
      box.innerHTML = '<span class="small" style="color:#656d76;">暂无活动房间</span>';
      renderGrid();
      return;
    }
    if (groupIdx >= groups.length) groupIdx = 0;
    groups.forEach((g, i) => {
      const b = document.createElement('button');
      b.textContent = `第 ${i + 1} 组（${g.length}）`;
      if (i === groupIdx) b.classList.add('active');
      b.addEventListener('click', () => { groupIdx = i; renderGroupTabs(); });
      box.appendChild(b);
    });
    renderGrid();
  }

  function renderGrid() {
    const box = $('gridBox');
    box.innerHTML = '';
    const cur = groups[groupIdx] || [];
    for (let i = 0; i < 4; i++) {
      const cell = document.createElement('div');
      cell.className = 'grid-cell' + (cur[i] ? '' : ' empty');
      if (cur[i]) {
        const label = document.createElement('div');
        label.className = 'label';
        label.textContent = cur[i].name;
        const v = document.createElement('video');
        v.autoplay = true; v.muted = true; v.controls = false;
        cell.appendChild(v);
        cell.appendChild(label);
        box.appendChild(cell);
        attachPlayer(v, cur[i].play_url, i, gridPlayers);
      } else {
        destroyPlayer(i, gridPlayers);
        box.appendChild(cell);
      }
    }
  }

  async function refresh() {
    try {
      if (mode === 'list') {
        const d = await fetchJSON('/admin/api/rooms');
        rooms = d.rooms || [];
        renderList();
      } else {
        const d = await fetchJSON('/admin/api/groups');
        groups = d.groups || [];
        renderGroupTabs();
      }
    } catch (_) {}
  }

  function tearDownPlayers(pool) {
    for (let i = 0; i < pool.length; i++) destroyPlayer(i, pool);
  }

  function switchMode(next) {
    if (next === mode) return;
    mode = next;
    $('tabList').classList.toggle('active', mode === 'list');
    $('tabGrid').classList.toggle('active', mode === 'grid');
    $('viewList').hidden = mode !== 'list';
    $('viewGrid').hidden = mode !== 'grid';
    // stop inactive players
    if (mode === 'list') {
      tearDownPlayers(gridPlayers);
    } else {
      tearDownPlayers(listPlayers);
    }
    refresh();
  }

  $('tabList').addEventListener('click', () => switchMode('list'));
  $('tabGrid').addEventListener('click', () => switchMode('grid'));
  $('btnLogout').addEventListener('click', async () => {
    await fetch('/admin/api/logout', { method: 'POST', credentials: 'same-origin' });
    location.href = '/admin/login.html';
  });

  refresh();
  pollTimer = setInterval(refresh, 5000);
})();
