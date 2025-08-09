<script>
  import { createEventDispatcher } from 'svelte'

  export let pdfUrl = ''
  export let pdfViewerKey = 0
  export let hasUnsavedChanges = false

  const dispatch = createEventDispatcher()

  function saveCurrentPdf() {
    dispatch('save-pdf')
  }
</script>

<div class="pdf-viewer-section">
  <div class="section-header pdf-header">
    <div class="pdf-title">
      <h3>PDFプレビュー</h3>
      {#if hasUnsavedChanges}
        <span class="unsaved-indicator">●未保存</span>
      {/if}
    </div>
    {#if pdfUrl}
      <div class="pdf-actions">
        <button class="btn-save" on:click={saveCurrentPdf} title="PDFファイルを保存">
          💾 保存
        </button>
      </div>
    {/if}
  </div>
  <div class="pdf-viewer-container">
    {#if pdfUrl}
      {#key pdfViewerKey}
        <embed src={pdfUrl} type="application/pdf" class="pdf-viewer" />
      {/key}
    {:else}
      <div class="pdf-placeholder">
        <div>
          <h3>PDFが生成されるとここに表示されます</h3>
          <p>左側でファイルを選択してPDFに変換してください。</p>
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  .pdf-viewer-section {
    display: flex;
    flex-direction: column;
    overflow: hidden;
    height: 100%;
  }

  .section-header {
    padding: 0.5rem 1rem;
    background: #f8f9fa;
    border-bottom: 1px solid #dee2e6;
    flex-shrink: 0;
  }

  .section-header h3 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    color: #495057;
  }

  /* PDF header with save functionality */
  .pdf-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .pdf-title {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .unsaved-indicator {
    color: #dc3545;
    font-size: 12px;
    font-weight: 500;
  }

  .pdf-actions {
    display: flex;
    gap: 0.5rem;
  }

  .btn-save {
    background: #28a745;
    color: white;
    border: none;
    padding: 0.375rem 0.75rem;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    transition: background-color 0.15s ease-in-out;
  }

  .btn-save:hover {
    background: #218838;
  }

  .btn-save:active {
    background: #1e7e34;
  }

  .pdf-viewer-container {
    flex: 1;
    overflow: hidden;
    background: #7f8b11;
    position: relative;
    min-height: 0;
    height: 100%;
  }

  .pdf-viewer {
    width: 100%;
    height: 100%;
    border: none;
  }

  .pdf-placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    background: white;
    color: #666;
    text-align: center;
  }

  .pdf-placeholder h3 {
    margin: 0 0 1rem 0;
    color: #495057;
  }

  .pdf-placeholder p {
    margin: 0;
    color: #6c757d;
  }
</style>
