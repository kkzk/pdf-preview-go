<script>
  import { createEventDispatcher } from 'svelte'
  import TreeNode from './TreeNode.svelte'

  export let fileTree = []
  export let selectedFiles = []
  export let expandedFolders = new Set()

  const dispatch = createEventDispatcher()

  function handleToggleFolder(event) {
    dispatch('toggle-folder', event.detail)
  }

  function handleToggleSelection(event) {
    dispatch('toggle-selection', event.detail)
  }
</script>

<div class="panel-section file-tree-section">
  <div class="section-header-compact">
    <h3>ファイル一覧</h3>
  </div>
  <div class="file-tree">
    {#if fileTree.length === 0}
      <div class="no-files">ディレクトリを読み込んでいます...</div>
    {:else}
      {#each fileTree as rootNode}
        <TreeNode
          node={rootNode}
          {selectedFiles}
          {expandedFolders}
          on:toggle-folder={handleToggleFolder}
          on:toggle-selection={handleToggleSelection}
        />
      {/each}
    {/if}
  </div>
</div>

<style>
  .panel-section {
    display: flex;
    flex-direction: column;
    overflow: hidden;
    position: relative;
    height: 100%; /* 親から与えられた高さを完全に使用 */
    min-height: 150px;
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

  .file-tree {
    flex: 1;
    overflow-y: auto;
    border: 1px solid #dee2e6;
    border-radius: 4px;
    background: white;
    min-height: 0; /* flexアイテムがshrinkできるように */
    max-height: 100%; /* 親コンテナを超えないように */
  }

  /* Empty state message */
  .no-files {
    flex: 1; /* 利用可能な領域を埋める */
    display: flex;
    align-items: center;
    justify-content: center;
    text-align: center;
    color: #6c757d;
    font-size: 12px;
    background: white;
    min-height: 120px; /* 最小高さを確保 */
  }
</style>
