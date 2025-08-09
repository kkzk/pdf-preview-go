<script>
  import { createEventDispatcher } from 'svelte'

  export let node
  export let selectedFiles = []
  export let expandedFolders = new Set()
  export let depth = 0

  const dispatch = createEventDispatcher()

  $: isSelected = selectedFiles.some(f => f.path === node.path)
  $: isExpanded = expandedFolders.has(node.path)
  $: hasChildren = node.children && node.children.length > 0

  function toggleExpanded() {
    if (node.isDir && hasChildren) {
      dispatch('toggle-folder', node.path)
    }
  }

  function toggleSelection() {
    if (!node.isDir) {
      dispatch('toggle-selection', node)
    }
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
  <div class="node-content" 
       class:folder={node.isDir} 
       class:file={!node.isDir}
       class:clickable={node.isDir ? hasChildren : true}
       on:click={node.isDir ? (hasChildren ? toggleExpanded : undefined) : toggleSelection}
       on:keydown={e => e.key === 'Enter' && (node.isDir ? (hasChildren && toggleExpanded()) : toggleSelection())}
       tabindex={node.isDir ? (hasChildren ? 0 : -1) : 0}
       role="button">
    {#if node.isDir}
      <button class="folder-toggle" on:click|stopPropagation={toggleExpanded} disabled={!hasChildren}>
        {getFileIcon(node)}
      </button>
      <span class="node-name folder-name" class:disabled={!hasChildren}>
        {node.name}
      </span>
      {#if hasChildren}
        <span class="child-count">({node.children.length})</span>
      {/if}
    {:else}
      <input type="checkbox" checked={isSelected} on:change|stopPropagation={toggleSelection} />
      <span class="file-icon">{getFileIcon(node)}</span>
      <span class="node-name">{node.name}</span>
      <span class="file-size">{formatFileSize(node.size)}</span>
    {/if}
  </div>

  {#if node.isDir && isExpanded && hasChildren}
    <div class="children">
      {#each node.children as child}
        <svelte:self
          node={child}
          {selectedFiles}
          {expandedFolders}
          depth={depth + 1}
          on:toggle-folder
          on:toggle-selection
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

  input[type='checkbox'] {
    accent-color: #007bff;
    width: 14px;
    height: 14px;
  }
</style>
