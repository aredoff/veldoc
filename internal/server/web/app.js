let pollInterval = 3000;
let selectedPath = null;
let selectedMeta = null;
let pollTimer = null;
let isMarkdown = false;
let isImage = false;
let treeNodes = [];
const expandedDirs = new Set();

const treeEl = document.getElementById('tree');
const emptyEl = document.getElementById('empty');
const viewerEl = document.getElementById('viewer');
const filePathEl = document.getElementById('file-path');
const rawView = document.getElementById('raw-view');
const mdView = document.getElementById('md-view');
const imageView = document.getElementById('image-view');
const imagePreview = document.getElementById('image-preview');
const modeRawBtn = document.getElementById('mode-raw');
const modeMdBtn = document.getElementById('mode-md');
const logoutForm = document.getElementById('logout-form');

async function init() {
  try {
    const res = await fetch('/api/config');
    const cfg = await res.json();
    pollInterval = cfg.pollIntervalMs || 3000;
    if (cfg.auth === 'form') {
      logoutForm.classList.remove('hidden');
    }
  } catch (_) {}

  await loadTree();
  pollTimer = setInterval(loadTree, pollInterval);

  const initialPath = getPathFromURL();
  if (initialPath) {
    await openPath(initialPath, { skipHistory: true, replaceHistory: true });
  }

  window.addEventListener('popstate', onPopState);

  document.getElementById('refresh-btn').addEventListener('click', loadTree);
  modeRawBtn.addEventListener('click', () => setMode('raw'));
  modeMdBtn.addEventListener('click', () => setMode('md'));
}

function getPathFromURL() {
  const path = new URLSearchParams(location.search).get('path');
  return path || null;
}

function setPathInURL(path, replace) {
  const url = '/?path=' + encodeURIComponent(path);
  const state = { path };
  if (replace) {
    history.replaceState(state, '', url);
  } else {
    history.pushState(state, '', url);
  }
}

async function onPopState(e) {
  const path = e.state?.path ?? getPathFromURL();
  if (path) {
    await openPath(path, { skipHistory: true });
  } else {
    clearSelection();
  }
}

function clearSelection() {
  selectedPath = null;
  selectedMeta = null;
  viewerEl.classList.add('hidden');
  emptyEl.classList.remove('hidden');
  document.title = 'Veldoc';
  document.querySelectorAll('.tree-item').forEach(el => {
    el.classList.remove('active');
  });
}

async function openPath(path, options = {}) {
  const node = findNode(treeNodes, path);
  if (!node || node.type !== 'file') {
    showNotFound(path);
    if (!options.skipHistory) {
      setPathInURL(path, options.replaceHistory);
    }
    return;
  }
  await selectFile(path, fileMeta(node), options);
}

async function loadTree() {
  try {
    const res = await fetch('/api/tree');
    if (!res.ok) return;
    const data = await res.json();
    const nodes = data.children || [];

    treeNodes = nodes;

    if (selectedPath) {
      const node = findNode(nodes, selectedPath);
      if (!node) {
        showNotFound(selectedPath);
      } else if (metaChanged(selectedMeta, node)) {
        selectedMeta = fileMeta(node);
        await loadFileContent(selectedPath, { preserveScroll: true });
      } else {
        selectedMeta = fileMeta(node);
      }
      ensureExpandedForPath(selectedPath);
    }

    renderTree(treeNodes, 0);
    if (selectedPath) {
      highlightSelected();
    }
  } catch (_) {}
}

function findNode(nodes, path) {
  for (const node of nodes) {
    if (node.path === path) {
      return node;
    }
    if (node.children) {
      const found = findNode(node.children, path);
      if (found) return found;
    }
  }
  return null;
}

function fileMeta(node) {
  return { modified: node.modified, size: node.size };
}

function metaChanged(prev, node) {
  if (!prev) return false;
  const next = fileMeta(node);
  return prev.modified !== next.modified || prev.size !== next.size;
}

function ensureExpandedForPath(path) {
  for (const dir of parentDirs(path)) {
    expandedDirs.add(dir);
  }
}

function parentDirs(path) {
  const parts = path.split('/');
  const dirs = [];
  for (let i = 0; i < parts.length - 1; i++) {
    dirs.push(parts.slice(0, i + 1).join('/'));
  }
  return dirs;
}

function toggleDir(path) {
  if (expandedDirs.has(path)) {
    expandedDirs.delete(path);
  } else {
    expandedDirs.add(path);
  }
  renderTree(treeNodes, 0);
  highlightSelected();
}

function renderTree(nodes, depth, parentEl) {
  const container = parentEl || treeEl;
  if (depth === 0) {
    treeEl.innerHTML = '';
  }

  for (const node of nodes) {
    const branch = document.createElement('div');
    branch.className = 'tree-branch';

    const el = document.createElement('div');
    el.className = 'tree-item ' + node.type;
    el.style.setProperty('--depth', depth);
    el.dataset.path = node.path;

    const hasChildren = node.type === 'dir' && node.children && node.children.length > 0;
    const expanded = hasChildren && expandedDirs.has(node.path);

    const icon = document.createElement('span');
    icon.className = 'tree-icon';
    if (node.type === 'dir') {
      icon.textContent = hasChildren ? (expanded ? '▾' : '▸') : '▸';
    } else {
      icon.textContent = '·';
    }

    const name = document.createElement('span');
    name.className = 'tree-name';
    name.textContent = node.name;
    name.title = node.path || node.name;

    el.appendChild(icon);
    el.appendChild(name);

    if (node.type === 'file') {
      el.addEventListener('click', () => selectFile(node.path, fileMeta(node)));
    } else if (hasChildren) {
      el.addEventListener('click', () => toggleDir(node.path));
    }

    branch.appendChild(el);

    if (hasChildren) {
      const childrenEl = document.createElement('div');
      childrenEl.className = 'tree-children';
      if (!expanded) {
        childrenEl.classList.add('collapsed');
      }
      branch.appendChild(childrenEl);
      renderTree(node.children, depth + 1, childrenEl);
    }

    container.appendChild(branch);
  }
}

function highlightSelected() {
  document.querySelectorAll('.tree-item').forEach(el => {
    el.classList.toggle('active', el.dataset.path === selectedPath);
  });
}

async function selectFile(path, meta, options = {}) {
  selectedPath = path;
  selectedMeta = meta || null;
  ensureExpandedForPath(path);
  renderTree(treeNodes, 0);
  highlightSelected();

  isMarkdown = /\.(md|markdown)$/i.test(path);
  isImage = /\.(png|jpe?g|gif|webp|svg|ico|bmp)$/i.test(path);
  modeRawBtn.classList.toggle('hidden', isImage);
  modeMdBtn.classList.toggle('hidden', !isMarkdown || isImage);

  emptyEl.classList.add('hidden');
  viewerEl.classList.remove('hidden');
  filePathEl.textContent = path;

  if (isImage) {
    setMode('image');
  } else {
    setMode(isMarkdown ? 'md' : 'raw');
  }

  const ok = await loadFileContent(path);
  if (!ok) {
    return;
  }

  document.title = path + ' — Veldoc';
  if (!options.skipHistory) {
    setPathInURL(path, options.replaceHistory);
  }
}

async function loadFileContent(path, options = {}) {
  const preserveScroll = options.preserveScroll === true;
  const scrollEl = getActiveView();
  const scrollTop = preserveScroll ? scrollEl.scrollTop : 0;

  try {
    const res = await fetch('/api/file?path=' + encodeURIComponent(path));
    if (res.status === 404) {
      showNotFound(path);
      return false;
    }
    if (!res.ok) {
      const err = await res.json();
      showFileError('Error: ' + (err.error || res.statusText));
      return false;
    }
    const data = await res.json();

    if (data.kind === 'image') {
      const cacheKey = selectedMeta?.modified || Date.now();
      imagePreview.onerror = () => showNotFound(path);
      imagePreview.src = data.url + '&v=' + encodeURIComponent(String(cacheKey));
      imagePreview.alt = path;
      rawView.textContent = '';
      mdView.innerHTML = '';
      return true;
    }

    imagePreview.onerror = null;

    if (rawView.textContent !== data.content) {
      rawView.classList.remove('not-found');
      rawView.textContent = data.content;
    }

    if (isMarkdown) {
      const mdRes = await fetch('/api/markdown?path=' + encodeURIComponent(path));
      if (mdRes.ok) {
        const mdData = await mdRes.json();
        if (mdView.innerHTML !== mdData.html) {
          mdView.innerHTML = mdData.html;
        }
      }
    } else {
      mdView.innerHTML = '';
    }

    if (preserveScroll) {
      scrollEl.scrollTop = scrollTop;
    }
    return true;
  } catch (_) {
    showFileError('Failed to load file');
    return false;
  }
}

function getActiveView() {
  if (!imageView.classList.contains('hidden')) return imageView;
  return rawView.classList.contains('hidden') ? mdView : rawView;
}

function showNotFound(path) {
  selectedPath = path;
  selectedMeta = null;
  ensureExpandedForPath(path);
  renderTree(treeNodes, 0);
  highlightSelected();

  emptyEl.classList.add('hidden');
  viewerEl.classList.remove('hidden');
  filePathEl.textContent = path;
  modeRawBtn.classList.remove('hidden');
  modeMdBtn.classList.add('hidden');
  setMode('raw');
  rawView.classList.add('not-found');
  rawView.textContent = '404 — File not found';
  mdView.innerHTML = '';
  imagePreview.removeAttribute('src');
  document.title = '404 — Veldoc';
}

function showFileError(message) {
  rawView.textContent = message;
  mdView.innerHTML = '';
  imagePreview.removeAttribute('src');
  setMode('raw');
}

function setMode(mode) {
  const isRaw = mode === 'raw';
  const isMd = mode === 'md';
  const isImg = mode === 'image';
  modeRawBtn.classList.toggle('active', isRaw);
  modeMdBtn.classList.toggle('active', isMd);
  rawView.classList.toggle('hidden', !isRaw);
  mdView.classList.toggle('hidden', !isMd);
  imageView.classList.toggle('hidden', !isImg);
}

init();
