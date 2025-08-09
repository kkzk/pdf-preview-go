<script>
  export let leftPanelWidth = 300
  export let rightPanelSplit = 70
  export let fileTreeHeight = 40
  export let selectedFilesHeight = 35
  export let sheetsHeight = 25

  let isResizingLeftPanel = false
  let isResizingRightPanel = false
  let isResizingFileTree = false
  let isResizingSelectedFiles = false

  export let showRightHandle = false

  export function startResizeLeftPanel(e) {
    isResizingLeftPanel = true
    e.preventDefault()
  }

  export function startResizeRightPanel(e) {
    isResizingRightPanel = true
    e.preventDefault()
  }

  export function startResizeFileTree(e) {
    isResizingFileTree = true
    e.preventDefault()
  }

  export function startResizeSelectedFiles(e) {
    isResizingSelectedFiles = true
    e.preventDefault()
  }

  export function handleMouseMove(e) {
    if (isResizingLeftPanel) {
      const containerRect = document.querySelector('.app-container')?.getBoundingClientRect()
      if (containerRect) {
        const newWidth = Math.max(250, Math.min(500, e.clientX - containerRect.left))
        leftPanelWidth = newWidth
        dispatch('resize-left-panel', newWidth)
      }
    }

    if (isResizingRightPanel) {
      const rightPanel = document.querySelector('.right-panel')
      const rightRect = rightPanel?.getBoundingClientRect()
      if (rightRect) {
        const relativeY = e.clientY - rightRect.top
        const newSplit = Math.max(30, Math.min(80, (relativeY / rightRect.height) * 100))
        rightPanelSplit = newSplit
        dispatch('resize-right-panel', newSplit)
      }
    }

    if (isResizingFileTree) {
      const leftPanel = document.querySelector('.left-panel')
      const leftRect = leftPanel?.getBoundingClientRect()
      if (leftRect) {
        const relativeY = e.clientY - leftRect.top - 10 // Account for header
        const panelHeight = leftRect.height - 10
        const newHeight = Math.max(20, Math.min(60, (relativeY / panelHeight) * 100))

        const remaining = 100 - newHeight
        const ratio = selectedFilesHeight / (selectedFilesHeight + sheetsHeight)

        fileTreeHeight = newHeight
        selectedFilesHeight = remaining * ratio
        sheetsHeight = remaining * (1 - ratio)

        dispatch('resize-file-tree', { fileTreeHeight, selectedFilesHeight, sheetsHeight })
      }
    }

    if (isResizingSelectedFiles) {
      const leftPanel = document.querySelector('.left-panel')
      const leftRect = leftPanel?.getBoundingClientRect()
      if (leftRect) {
        const relativeY = e.clientY - leftRect.top - 10
        const panelHeight = leftRect.height - 10
        const treeBottom = (fileTreeHeight / 100) * panelHeight
        const availableHeight = panelHeight - treeBottom
        const newSelectedHeight = Math.max(
          15,
          Math.min(60, ((relativeY - treeBottom) / availableHeight) * 100)
        )

        const totalRemaining = 100 - fileTreeHeight
        selectedFilesHeight = (newSelectedHeight / 100) * totalRemaining
        sheetsHeight = totalRemaining - selectedFilesHeight

        dispatch('resize-selected-files', { selectedFilesHeight, sheetsHeight })
      }
    }
  }

  export function handleMouseUp() {
    isResizingLeftPanel = false
    isResizingRightPanel = false
    isResizingFileTree = false
    isResizingSelectedFiles = false
  }
</script>

<!-- Resize Handles -->
<div class="resize-handle vertical" on:mousedown={startResizeLeftPanel}></div>
{#if showRightHandle}
  <div class="resize-handle horizontal" on:mousedown={startResizeRightPanel}></div>
{/if}
<div class="resize-handle horizontal" on:mousedown={startResizeFileTree}></div>
<div class="resize-handle horizontal" on:mousedown={startResizeSelectedFiles}></div>

<style>
  /* Resize handles */
  .resize-handle {
    background: #dee2e6;
    position: relative;
    user-select: none;
    transition: background-color 0.2s ease;
  }

  .resize-handle:hover {
    background: #adb5bd;
  }

  .resize-handle.vertical {
    width: 4px;
    cursor: ew-resize;
    flex-shrink: 0;
  }

  .resize-handle.horizontal {
    height: 4px;
    cursor: ns-resize;
    flex-shrink: 0;
    margin: 0;
  }
</style>
