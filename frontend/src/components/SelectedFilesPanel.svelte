<script>
  import { createEventDispatcher } from 'svelte'

  /** @type {any[]} */
  export let selectedFiles = []
  /** @type {any} */
  export let currentFile = null

  const dispatch = createEventDispatcher()

  function selectFileFromList(file) {
    dispatch('select-file', file)
  }

  function moveFileUp(index) {
    if (index > 0) {
      dispatch('move-file', { from: index, to: index - 1 })
    }
  }

  function moveFileDown(index) {
    if (index < selectedFiles.length - 1) {
      dispatch('move-file', { from: index, to: index + 1 })
    }
  }

  function removeFile(index) {
    dispatch('remove-file', index)
  }
</script>

<div class="panel-section selected-files-section">
  <div class="section-header-compact">
    <h3>選択ファイル</h3>
    <span class="count-badge">({selectedFiles.length})</span>
  </div>
  <div class="selected-files">
    {#each selectedFiles as file, index}
      <div
        class="selected-file-item"
        class:active={currentFile && currentFile.path === file.path}
        on:click={() => selectFileFromList(file)}
        on:keydown={e => e.key === 'Enter' && selectFileFromList(file)}
        tabindex="0"
        role="button"
      >
        <div class="file-info">
          <span class="file-icon">
            {#if file.name.includes('.xls')}📊{:else if file.name.endsWith('.pdf')}📄{:else}📝{/if}
          </span>
          <span class="file-name">{file.name}</span>
        </div>
        <div class="file-controls">
          <button
            class="btn-small"
            on:click|stopPropagation={() => moveFileUp(index)}
            disabled={index === 0}>↑</button
          >
          <button
            class="btn-small"
            on:click|stopPropagation={() => moveFileDown(index)}
            disabled={index === selectedFiles.length - 1}>↓</button
          >
          <button class="btn-small btn-danger" on:click|stopPropagation={() => removeFile(index)}
            >×</button
          >
        </div>
      </div>
    {/each}
  </div>
</div>

<style>
  .panel-section {
    display: flex;
    flex-direction: column;
    overflow: hidden;
    position: relative;
    min-height: 100px;
  }

  .section-header-compact {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.25rem;
    flex-shrink: 0;
  }

  .section-header-compact h3 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    color: #495057;
  }

  .count-badge {
    background: #007bff;
    color: white;
    font-size: 11px;
    padding: 0.125rem 0.375rem;
    border-radius: 10px;
    font-weight: 500;
  }

  .selected-files {
    flex: 1;
    overflow-y: auto;
    border: 1px solid #dee2e6;
    border-radius: 4px;
    background: white;
    min-height: 0;
  }

  .selected-file-item {
    display: flex;
    align-items: center;
    padding: 0.5rem;
    border-bottom: 1px solid #f8f9fa;
    gap: 0.5rem;
    background: white;
    cursor: pointer;
    border-radius: 4px;
    transition: background-color 0.2s ease;
    user-select: none;
  }

  .selected-file-item:hover {
    background: #f8f9fa;
  }

  .selected-file-item.active {
    background: #e7f3ff;
    border-color: #007bff;
  }

  .selected-file-item:focus {
    outline: 2px solid rgba(0, 123, 255, 0.25);
    outline-offset: -2px;
  }

  .file-info {
    display: flex;
    align-items: center;
    flex: 1;
    gap: 0.5rem;
  }

  .file-info .file-name {
    font-size: 12px;
    color: #495057;
  }

  .file-controls {
    display: flex;
    gap: 0.25rem;
  }

  .btn-small {
    padding: 0.25rem;
    font-size: 10px;
    border: 1px solid #ddd;
    background: white;
    color: #495057;
    border-radius: 2px;
    cursor: pointer;
    min-width: 20px;
  }

  .btn-small:hover:not(:disabled) {
    background: #f8f9fa;
    color: #212529;
  }

  .btn-small:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-danger {
    background: #dc3545;
    color: white;
    border-color: #dc3545;
  }

  .btn-danger:hover {
    background: #c82333;
  }
</style>
