<script>
  export let logs = []
  export let isLogExpanded = false
  export let effectiveRightPanelSplit = 95

  function toggleLogExpanded() {
    isLogExpanded = !isLogExpanded

    // Scroll to bottom of logs if expanded
    if (isLogExpanded) {
      setTimeout(() => {
        const logContainer = document.querySelector('.log-container')
        if (logContainer) {
          logContainer.scrollTop = logContainer.scrollHeight
        }
      }, 100)
    }
  }
</script>

<div
  class="log-section"
  style={isLogExpanded
    ? `height: ${100 - effectiveRightPanelSplit}%; flex-shrink: 0;`
    : 'height: auto; flex-shrink: 0;'}
>
  <div
    class="section-header clickable"
    on:click={toggleLogExpanded}
    on:keydown={e => e.key === 'Enter' && toggleLogExpanded()}
    tabindex="0"
    role="button"
  >
    <h3>ログ</h3>
    <span class="toggle-icon">{isLogExpanded ? '▼' : '▶'}</span>
  </div>
  {#if isLogExpanded}
    <div class="log-container">
      {#each logs as log}
        <div class="log-entry">{log}</div>
      {/each}
    </div>
  {:else}
    <div class="log-collapsed">
      <div class="log-summary">
        {logs.length > 0 ? `最新: ${logs[logs.length - 1]}` : 'ログなし'}
      </div>
    </div>
  {/if}
</div>

<style>
  .log-section {
    display: flex;
    flex-direction: column;
    min-height: 30px; /* 最小高さを縮小してPDF表示エリアを広く */
  }

  .section-header {
    padding: 0.5rem 1rem;
    background: #f8f9fa;
    border-bottom: 1px solid #dee2e6;
    flex-shrink: 0;
  }

  .section-header.clickable {
    cursor: pointer;
    user-select: none;
    display: flex;
    align-items: center;
    justify-content: space-between;
    outline: none;
  }

  .section-header.clickable:hover {
    background: #e9ecef;
  }

  .section-header.clickable:focus {
    background: #e9ecef;
    box-shadow: 0 0 0 2px rgba(0, 123, 255, 0.25);
  }

  .toggle-icon {
    font-size: 12px;
    color: #6c757d;
    transition: transform 0.2s ease;
  }

  .section-header h3 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    color: #495057;
  }

  .log-container {
    flex: 1;
    overflow-y: auto;
    padding: 0.5rem;
    background: #f8f9fa;
    color: #495057;
    font-family: 'Consolas', 'Monaco', monospace;
    font-size: 11px;
    line-height: 1.4;
    text-align: left;
  }

  .log-entry {
    margin-bottom: 0.25rem;
    word-break: break-word;
    padding: 0.125rem 0;
  }

  .log-collapsed {
    flex: 1;
    display: flex;
    align-items: center;
    padding: 0.25rem 1rem;
    background: #f8f9fa;
    border-top: 1px solid #dee2e6;
    min-height: 28px; /* ヘッダー高さを縮小 */
  }

  .log-summary {
    color: #6c757d;
    font-size: 11px;
    font-style: italic;
    text-align: left;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 100%;
    line-height: 1.2;
  }
</style>
