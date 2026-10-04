<script>
  import { createEventDispatcher } from 'svelte'

  export let node
  export let selectedFiles = []
  export let expandedFolders = new Set()
  export let currentPath = '' // path of the file whose sheets are shown
  export let depth = 0

  const dispatch = createEventDispatcher()

  $: isSelected = selectedFiles.some(f => f.path === node.path)
  $: isExpanded = expandedFolders.has(node.path)
  $: isCurrent = !node.isDir && node.path === currentPath
  // Folder contents are loaded when the folder is first expanded:
  // children is undefined until then, and [] for an empty folder
  $: loaded = Array.isArray(node.children)
  $: isEmpty = loaded && node.children.length === 0
  $: canExpand = node.isDir && !isEmpty

  function toggleExpanded() {
    if (canExpand) {
      dispatch('toggle-folder', node.path)
    }
  }

  function toggleSelection() {
    if (!node.isDir) {
      dispatch('toggle-selection', node)
    }
  }

  // Clicking a file row shows the file (its sheets) without changing which
  // files are selected for conversion; only the checkbox changes that
  function showFile() {
    if (!node.isDir) {
      dispatch('select-file', node)
    }
  }

  /** @param {KeyboardEvent} e */
  function handleKeydown(e) {
    if (e.target !== e.currentTarget) {
      return // e.g. Space on the checkbox itself
    }
    if (node.isDir) {
      if (e.key === 'Enter' && canExpand) toggleExpanded()
    } else if (e.key === 'Enter') {
      showFile()
    } else if (e.key === ' ') {
      e.preventDefault()
      toggleSelection()
    }
  }

  function openFile() {
    dispatch('open-file', node)
  }

  function getFileIcon(node) {
    if (node.isDir) {
      return isExpanded ? '📂' : '📁'
    }
    if (node.name.endsWith('.pdf')) return '📄'
    if (node.name.includes('.xls')) return '📊'
    return '📝'
  }

  function formatFileSize(bytes) {
    if (bytes < 1024) return bytes + ' B'
    if (bytes < 1024 * 1024) return Math.round(bytes / 1024) + ' KB'
    return Math.round(bytes / (1024 * 1024)) + ' MB'
  }
</script>

<div class="tree-node" style="padding-left: {depth * 20}px">
  <div
    class="node-content"
    class:folder={node.isDir}
    class:file={!node.isDir}
    class:clickable={node.isDir ? canExpand : true}
    class:current={isCurrent}
    on:click={node.isDir ? (canExpand ? toggleExpanded : undefined) : showFile}
    on:keydown={handleKeydown}
    tabindex={node.isDir ? (canExpand ? 0 : -1) : 0}
    role="button"
  >
    {#if node.isDir}
      <button
        class="folder-toggle"
        on:click|stopPropagation={toggleExpanded}
        disabled={!canExpand}
      >
        {getFileIcon(node)}
      </button>
      <span class="node-name folder-name" class:disabled={isEmpty}>
        {node.name}
      </span>
      {#if isEmpty}
        <span class="child-count">(空)</span>
      {:else if loaded}
        <span class="child-count">({node.children.length})</span>
      {/if}
    {:else}
      <input
        type="checkbox"
        checked={isSelected}
        on:click|stopPropagation
        on:change={toggleSelection}
        title="PDFに変換するファイルとして選択"
      />
      <span class="file-icon">{getFileIcon(node)}</span>
      <span class="node-name">{node.name}</span>
      <span class="file-size">{formatFileSize(node.size)}</span>
      <button class="open-file" on:click|stopPropagation={openFile} title="アプリで開く">
        開く
      </button>
    {/if}
  </div>

  {#if node.isDir && isExpanded && node.loading}
    <div class="children">
      <div class="loading" style="padding-left: {(depth + 1) * 20}px">読み込み中…</div>
    </div>
  {:else if node.isDir && isExpanded && loaded && !isEmpty}
    <div class="children">
      {#each node.children as child}
        <svelte:self
          node={child}
          {selectedFiles}
          {expandedFolders}
          {currentPath}
          depth={depth + 1}
          on:toggle-folder
          on:toggle-selection
          on:select-file
          on:open-file
        />
      {/each}
    </div>
  {/if}
</div>

<style>
  .tree-node {
    user-select: none;
  }

  .node-content {
    display: flex;
    align-items: center;
    padding: 0.4rem 0.5rem;
    gap: 0.5rem;
    min-height: 32px;
    border-radius: 4px;
    transition: background-color 0.2s ease;
  }

  .node-content.clickable {
    cursor: pointer;
    user-select: none;
  }

  .node-content:hover {
    background: #f8f9fa;
  }

  .node-content.clickable:hover {
    background: #e9ecef;
  }

  /* The file whose sheets are shown */
  .node-content.current,
  .node-content.current:hover {
    background: #e7f1ff;
  }

  .node-content:focus {
    outline: 2px solid rgba(0, 123, 255, 0.25);
    outline-offset: -2px;
  }

  .folder-toggle {
    background: none;
    border: none;
    font-size: 16px;
    cursor: pointer;
    padding: 0;
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: flex-start;
    border-radius: 4px;
  }

  .folder-toggle:hover:not(:disabled) {
    background: #e9ecef;
  }

  .folder-toggle:disabled {
    opacity: 0.5;
    cursor: default;
  }

  .file-icon {
    font-size: 16px;
    width: 20px;
    display: flex;
    justify-content: flex-start;
  }

  .node-name {
    flex: 1;
    font-size: 13px;
    color: #495057;
    word-break: break-all;
    text-align: left;
  }

  .folder .node-name {
    font-weight: 500;
    color: #343a40;
  }

  .folder-name.disabled {
    color: #6c757d;
  }

  .child-count {
    font-size: 11px;
    color: #6c757d;
    background: #f8f9fa;
    padding: 0.1rem 0.3rem;
    border-radius: 10px;
  }

  .file-size {
    font-size: 11px;
    color: #6c757d;
    min-width: 60px;
  }

  .children {
    border-left: 1px dotted #dee2e6;
    margin-left: 12px;
  }

  .loading {
    font-size: 12px;
    color: #6c757d;
    padding-top: 0.25rem;
    padding-bottom: 0.25rem;
  }

  .open-file {
    visibility: hidden;
    flex-shrink: 0;
    padding: 0.1rem 0.4rem;
    border: 1px solid #ced4da;
    border-radius: 4px;
    background: white;
    color: #495057;
    font-size: 11px;
    cursor: pointer;
  }

  /* Show the open button on hover or keyboard focus */
  .node-content:hover .open-file,
  .node-content:focus-within .open-file {
    visibility: visible;
  }

  .open-file:hover {
    background: #e9ecef;
  }

  input[type='checkbox'] {
    accent-color: #007bff;
    width: 14px;
    height: 14px;
  }
</style>
