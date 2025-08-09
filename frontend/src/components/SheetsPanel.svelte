<script>
  import { createEventDispatcher } from 'svelte'

  /** @type {any} */
  export let currentFile = null
  /** @type {any[]} */
  export let excelSheets = []
  /** @type {Record<string, string[]>} */
  export let sheetSelections = {}
  /** @type {any[]} */
  export let selectedFiles = []
  export let isConverting = false
  export let autoUpdateEnabled = true

  const dispatch = createEventDispatcher()

  function isSheetSelected(sheetName) {
    if (!currentFile) return false
    const selections = sheetSelections[currentFile.path] || []
    return selections.includes(sheetName)
  }

  function toggleSheetSelection(sheetName) {
    dispatch('toggle-sheet', sheetName)
  }

  function convertToPDF() {
    dispatch('convert-pdf')
  }

  function toggleAutoUpdate() {
    dispatch('toggle-auto-update')
  }
</script>

<div class="panel-section sheets-section">
  <div class="section-header-compact">
    <h3>シート選択</h3>
    {#if currentFile}
      <span class="file-badge">{currentFile.name}</span>
    {/if}
  </div>
  <div class="sheets-content">
    {#if currentFile && excelSheets.length > 0}
      <div class="sheets-list">
        {#each excelSheets as sheet}
          <label class="sheet-checkbox" class:disabled={!sheet.visible}>
            <input
              type="checkbox"
              disabled={!sheet.visible}
              checked={isSheetSelected(sheet.name)}
              on:change={() => toggleSheetSelection(sheet.name)}
            />
            <span class="sheet-name">{sheet.name}</span>
            {#if !sheet.visible}<span class="sheet-hidden">(非表示)</span>{/if}
          </label>
        {/each}
      </div>
    {:else}
      <div class="no-sheets">Excelファイルを選択してください</div>
    {/if}
  </div>

  <!-- Convert Button -->
  <div class="convert-section">
    <button
      class="btn-primary btn-large"
      on:click={convertToPDF}
      disabled={selectedFiles.length === 0 || isConverting}
    >
      {#if isConverting}変換中...{:else}📄 PDFに変換{/if}
    </button>

    <!-- Auto-update toggle -->
    <div class="auto-update-section">
      <label class="auto-update-checkbox">
        <input type="checkbox" bind:checked={autoUpdateEnabled} on:change={toggleAutoUpdate} />
        <span class="auto-update-label">ファイル変更時に自動更新</span>
      </label>
    </div>
  </div>
</div>

<style>
  .panel-section {
    display: flex;
    flex-direction: column;
    overflow: hidden;
    position: relative;
    min-height: 80px;
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

  .file-badge {
    background: #28a745;
    color: white;
    font-size: 10px;
    padding: 0.125rem 0.375rem;
    border-radius: 8px;
    font-weight: 500;
    max-width: 150px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .sheets-content {
    flex: 1;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .sheets-list {
    flex: 1;
    overflow-y: auto;
    border: 1px solid #dee2e6;
    border-radius: 4px;
    background: white;
    min-height: 0;
  }

  .convert-section {
    margin-top: auto;
    padding-top: 0.25rem;
    flex-shrink: 0;
  }

  .sheet-checkbox {
    display: flex;
    align-items: center;
    padding: 0.5rem;
    cursor: pointer;
    gap: 0.5rem;
    border-bottom: 1px solid #f8f9fa;
    background: white;
  }

  .sheet-checkbox:hover:not(.disabled) {
    background: #f8f9fa;
  }

  .sheet-checkbox.disabled {
    opacity: 0.5;
    cursor: not-allowed;
    background: #f8f9fa;
  }

  .sheet-name {
    font-size: 12px;
    color: #495057;
  }

  .sheet-hidden {
    font-size: 11px;
    color: #6c757d;
  }

  .no-sheets {
    padding: 1rem;
    text-align: center;
    color: #6c757d;
    font-size: 12px;
    background: white;
  }

  /* Input elements styling */
  input[type='checkbox'] {
    margin-right: 0.5rem;
    accent-color: #007bff;
  }

  /* Text color improvements */
  label {
    color: #495057;
  }

  /* Buttons */
  .btn-primary {
    padding: 0.5rem 1rem;
    background: #007bff;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-size: 12px;
  }

  .btn-primary:hover:not(:disabled) {
    background: #0056b3;
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-large {
    width: 100%;
    padding: 0.75rem;
    font-size: 14px;
    font-weight: 600;
  }

  /* Auto-update section */
  .auto-update-section {
    margin-top: 0.5rem;
  }

  .auto-update-checkbox {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 12px;
    color: #495057;
    cursor: pointer;
  }

  .auto-update-checkbox input[type='checkbox'] {
    accent-color: #007bff;
    width: 14px;
    height: 14px;
  }

  .auto-update-label {
    user-select: none;
  }
</style>
