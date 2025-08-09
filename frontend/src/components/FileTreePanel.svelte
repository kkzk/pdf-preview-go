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
    {#each fileTree as rootNode}
      <TreeNode
        node={rootNode}
        {selectedFiles}
        {expandedFolders}
        on:toggle-folder={handleToggleFolder}
        on:toggle-selection={handleToggleSelection}
      />
    {/each}
  </div>
</div>

<style>
  .panel-section {
    display: flex;
    flex-direction: column;
    overflow: hidden;
    position: relative;
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
    min-height: 0;
  }
</style>
