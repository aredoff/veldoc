let pollInterval = 3000;
let selectedPath = null;
let selectedMeta = null;
let pollTimer = null;
let isMarkdown = false;
let treeNodes = [];
const expandedDirs = new Set();
let previewObjectURL = null;
let loadRequestId = 0;

const treeEl = document.getElementById('tree');
const emptyEl = document.getElementById('empty');
const viewerEl = document.getElementById('viewer');
const filePathEl = document.getElementById('file-path');
const rawView = document.getElementById('raw-view');
const mdView = document.getElementById('md-view');
const previewView = document.getElementById('preview-view');
const previewContent = document.getElementById('preview-content');
const downloadBtn = document.getElementById('download-btn');
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
  updateDownloadLink(null);
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

    const isDir = node.type === 'dir';
    const hasChildren = isDir && node.children && node.children.length > 0;
    const expanded = isDir && expandedDirs.has(node.path);

    const icon = document.createElement('span');
    icon.className = 'tree-icon';
    if (isDir) {
      icon.textContent = expanded ? '▾' : '▸';
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
    } else if (isDir) {
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

  modeRawBtn.classList.add('hidden');
  modeMdBtn.classList.add('hidden');

  emptyEl.classList.add('hidden');
  viewerEl.classList.remove('hidden');
  filePathEl.textContent = path;
  updateDownloadLink(path);

  const ok = await loadFileContent(path);
  if (!ok || selectedPath !== path) {
    return;
  }

  document.title = path + ' — Veldoc';
  if (!options.skipHistory) {
    setPathInURL(path, options.replaceHistory);
  }
}

function clearPreview() {
  if (previewObjectURL) {
    URL.revokeObjectURL(previewObjectURL);
    previewObjectURL = null;
  }
  previewContent.innerHTML = '';
}

function fileNameFromPath(path) {
  const parts = path.split('/');
  return parts[parts.length - 1] || 'download';
}

function downloadURL(path) {
  let url = '/api/raw?path=' + encodeURIComponent(path) + '&download=1';
  if (selectedMeta?.modified) {
    url += '&v=' + encodeURIComponent(selectedMeta.modified);
  }
  return url;
}

function updateDownloadLink(path) {
  if (!path) {
    downloadBtn.classList.add('hidden');
    downloadBtn.removeAttribute('href');
    downloadBtn.removeAttribute('download');
    return;
  }
  downloadBtn.href = downloadURL(path);
  downloadBtn.download = fileNameFromPath(path);
  downloadBtn.classList.remove('hidden');
}

function previewURL(baseUrl) {
  const cacheKey = selectedMeta?.modified || Date.now();
  return baseUrl + '&v=' + encodeURIComponent(String(cacheKey));
}

async function fetchPreviewBlob(url) {
  const res = await fetch(url, { credentials: 'same-origin' });
  if (!res.ok) {
    let message = res.statusText;
    try {
      const err = await res.json();
      if (err.error) {
        message = err.error;
      }
    } catch (_) {}
    throw new Error(message);
  }
  return res.blob();
}

async function showBinaryPreview(url, mime, requestId) {
  if (requestId !== loadRequestId) {
    return false;
  }
  clearPreview();
  const src = previewURL(url);

  const appendMedia = (el, blobSrc) => {
    if (el.tagName === 'OBJECT') {
      el.data = blobSrc;
    } else {
      el.src = blobSrc;
    }
    previewContent.appendChild(el);
  };

  try {
    const blob = await fetchPreviewBlob(src);
    if (requestId !== loadRequestId) {
      return false;
    }
    previewObjectURL = URL.createObjectURL(blob);
    if (requestId !== loadRequestId) {
      URL.revokeObjectURL(previewObjectURL);
      previewObjectURL = null;
      return false;
    }
    const blobSrc = previewObjectURL;

    if (mime.startsWith('image/')) {
      const el = document.createElement('img');
      el.alt = selectedPath || '';
      appendMedia(el, blobSrc);
      return true;
    }

    if (mime.startsWith('video/')) {
      const el = document.createElement('video');
      el.controls = true;
      appendMedia(el, blobSrc);
      return true;
    }

    if (mime.startsWith('audio/')) {
      const el = document.createElement('audio');
      el.controls = true;
      appendMedia(el, blobSrc);
      return true;
    }

    let el;
    if (mime === 'application/pdf') {
      el = document.createElement('embed');
      el.type = 'application/pdf';
    } else {
      el = document.createElement('object');
      el.type = mime;
    }
    appendMedia(el, blobSrc);
    return true;
  } catch (e) {
    showFileError('Error: ' + e.message);
    return false;
  }
}

async function loadFileContent(path, options = {}) {
  const requestId = ++loadRequestId;
  const isCurrent = () => requestId === loadRequestId;
  const preserveScroll = options.preserveScroll === true;
  const scrollEl = getActiveView();
  const scrollTop = preserveScroll ? scrollEl.scrollTop : 0;

  try {
    const res = await fetch('/api/file?path=' + encodeURIComponent(path));
    if (!isCurrent()) {
      return true;
    }
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
    if (!isCurrent()) {
      return true;
    }

    if (data.kind === 'binary' || data.kind === 'image') {
      modeRawBtn.classList.add('hidden');
      modeMdBtn.classList.add('hidden');
      setMode('preview');
      const previewOk = await showBinaryPreview(
        data.url,
        data.mime || 'application/octet-stream',
        requestId,
      );
      if (!isCurrent()) {
        return true;
      }
      if (!previewOk) {
        return false;
      }
      rawView.textContent = '';
      mdView.innerHTML = '';
      return true;
    }

    modeRawBtn.classList.remove('hidden');
    isMarkdown = /\.(md|markdown)$/i.test(path);
    modeMdBtn.classList.toggle('hidden', !isMarkdown);
    if (!preserveScroll || !previewView.classList.contains('hidden')) {
      setMode(isMarkdown ? 'md' : 'raw');
    }
    clearPreview();

    if (rawView.textContent !== data.content) {
      rawView.classList.remove('not-found');
      rawView.textContent = data.content;
    }

    if (isMarkdown) {
      const mdRes = await fetch('/api/markdown?path=' + encodeURIComponent(path));
      if (!isCurrent()) {
        return true;
      }
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
    if (!isCurrent()) {
      return true;
    }
    showFileError('Failed to load file');
    return false;
  }
}

function getActiveView() {
  if (!previewView.classList.contains('hidden')) return previewView;
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
  updateDownloadLink(null);
  modeRawBtn.classList.remove('hidden');
  modeMdBtn.classList.add('hidden');
  setMode('raw');
  clearPreview();
  rawView.classList.add('not-found');
  rawView.textContent = '404 — File not found';
  mdView.innerHTML = '';
  document.title = '404 — Veldoc';
}

function showFileError(message) {
  clearPreview();
  rawView.textContent = message;
  mdView.innerHTML = '';
  modeRawBtn.classList.remove('hidden');
  modeMdBtn.classList.add('hidden');
  setMode('raw');
}

function setMode(mode) {
  const isRaw = mode === 'raw';
  const isMd = mode === 'md';
  const isPreview = mode === 'preview';
  modeRawBtn.classList.toggle('active', isRaw);
  modeMdBtn.classList.toggle('active', isMd);
  rawView.classList.toggle('hidden', !isRaw);
  mdView.classList.toggle('hidden', !isMd);
  previewView.classList.toggle('hidden', !isPreview);
}

init();
