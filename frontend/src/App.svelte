<script>
  import {OpenDirectoryDialog, GetDirectoryContents, GetDirectoryTree, GetExcelSheets, ConvertToPDF, GetFileInfo, GetInitialDirectory, SetWindowTitle} from '../wailsjs/go/main/App.js'
  import {onMount, onDestroy} from 'svelte'
  import {EventsOn, EventsOff} from '../wailsjs/runtime/runtime.js'
  import TreeNode from './TreeNode.svelte'

  // Main application state
  let rootDirectory = ''
  let fileTree = []
  let selectedFiles = []
  let currentFile = null
  let excelSheets = []
  let sheetSelections = {}
  let pdfUrl = ''
  let logs = []
  let isConverting = false

  // UI state
  let leftPanelWidth = 300
  let rightPanelSplit = 70 // percentage for PDF viewer when log is expanded
  let expandedFolders = new Set() // Track which folders are expanded
  let isLogExpanded = false // Track log section state
  
  // Dynamic panel split based on log state
  $: effectiveRightPanelSplit = isLogExpanded ? rightPanelSplit : 95
  
  // Left panel section heights (percentages)
  let fileTreeHeight = 40
  let selectedFilesHeight = 35
  let sheetsHeight = 25
  
  // Resize states
  let isResizingLeftPanel = false
  let isResizingRightPanel = false
  let isResizingFileTree = false
  let isResizingSelectedFiles = false

  // Initialize component
  onMount(async () => {
    try {
      // Get initial directory from command line argument
      const initialDir = await GetInitialDirectory()
      if (initialDir) {
        rootDirectory = initialDir
        await loadFileTree()
        await SetWindowTitle(initialDir)
        addLog(`作業ディレクトリを設定しました: ${initialDir}`)
      }
    } catch (error) {
      addLog(`作業ディレクトリ取得エラー: ${error}`)
    }

    // Listen for directory change events from menu
    EventsOn('directory-changed', async (newDir) => {
      rootDirectory = newDir
      expandedFolders.clear()
      expandedFolders = new Set()
      await loadFileTree()
      await SetWindowTitle(newDir)
      addLog(`作業フォルダを変更しました: ${newDir}`)
    })
  })

  onDestroy(() => {
    // Clean up event listeners
    EventsOff('directory-changed')
  })

  async function loadFileTree() {
    try {
      fileTree = await GetDirectoryTree(rootDirectory)
      addLog(`フォルダを読み込みました: ${rootDirectory}`)
    } catch (error) {
      // Fallback to flat directory listing if tree fails
      try {
        fileTree = await GetDirectoryContents(rootDirectory)
        addLog(`フォルダを読み込みました (フラット表示): ${rootDirectory}`)
      } catch (fallbackError) {
        addLog(`フォルダ読み込みエラー: ${error}`)
      }
    }
  }

  function toggleFolder(folderPath) {
    if (expandedFolders.has(folderPath)) {
      expandedFolders.delete(folderPath)
    } else {
      expandedFolders.add(folderPath)
    }
    expandedFolders = new Set(expandedFolders) // Trigger reactivity
  }

  function isFolderExpanded(folderPath) {
    return expandedFolders.has(folderPath)
  }

  function handleToggleFolder(event) {
    toggleFolder(event.detail)
  }

  function handleToggleSelection(event) {
    toggleFileSelection(event.detail)
  }

  function toggleFileSelection(file) {
    const index = selectedFiles.findIndex(f => f.path === file.path)
    if (index >= 0) {
      selectedFiles.splice(index, 1)
    } else {
      selectedFiles.push({...file})
    }
    selectedFiles = [...selectedFiles]
    
    // If it's an Excel file, load its sheets
    if (file.name.endsWith('.xlsx') || file.name.endsWith('.xlsm')) {
      loadExcelSheets(file)
    }
    
    addLog(`ファイル選択更新: ${file.name}`)
  }

  function isFileSelected(file) {
    return selectedFiles.some(f => f.path === file.path)
  }

  async function loadExcelSheets(file) {
    try {
      currentFile = file
      excelSheets = await GetExcelSheets(file.path)
      
      // Initialize sheet selections if not exists
      if (!sheetSelections[file.path]) {
        sheetSelections[file.path] = excelSheets
          .filter(sheet => sheet.visible)
          .map(sheet => sheet.name)
      }
      
      addLog(`Excelシートを読み込みました: ${file.name}`)
    } catch (error) {
      addLog(`Excelシート読み込みエラー: ${error}`)
    }
  }

  function toggleSheetSelection(sheetName) {
    if (!currentFile) return
    
    const filePath = currentFile.path
    if (!sheetSelections[filePath]) {
      sheetSelections[filePath] = []
    }
    
    const index = sheetSelections[filePath].indexOf(sheetName)
    if (index >= 0) {
      sheetSelections[filePath].splice(index, 1)
    } else {
      sheetSelections[filePath].push(sheetName)
    }
    
    sheetSelections = {...sheetSelections}
    addLog(`シート選択更新: ${sheetName}`)
  }

  function isSheetSelected(sheetName) {
    if (!currentFile) return false
    const selections = sheetSelections[currentFile.path] || []
    return selections.includes(sheetName)
  }

  function moveFileUp(index) {
    if (index > 0) {
      const temp = selectedFiles[index]
      selectedFiles[index] = selectedFiles[index - 1]
      selectedFiles[index - 1] = temp
      selectedFiles = [...selectedFiles]
      addLog('ファイル順序を変更しました')
    }
  }

  function moveFileDown(index) {
    if (index < selectedFiles.length - 1) {
      const temp = selectedFiles[index]
      selectedFiles[index] = selectedFiles[index + 1]
      selectedFiles[index + 1] = temp
      selectedFiles = [...selectedFiles]
      addLog('ファイル順序を変更しました')
    }
  }

  function removeFile(index) {
    const removed = selectedFiles.splice(index, 1)[0]
    selectedFiles = [...selectedFiles]
    addLog(`ファイルを削除しました: ${removed.name}`)
  }

  function selectFileFromList(file) {
    if (file.name.endsWith('.xlsx') || file.name.endsWith('.xlsm')) {
      loadExcelSheets(file)
    }
  }

  async function convertToPDF() {
    if (selectedFiles.length === 0) {
      addLog('変換するファイルが選択されていません')
      return
    }
    
    isConverting = true
    addLog('PDF変換を開始します...')
    
    try {
      const filePaths = selectedFiles.map(f => f.path)
      // Ensure sheetSelections has the correct type structure
      const validSheetSelections = Object.keys(sheetSelections).length > 0 
        ? sheetSelections 
        : Object.fromEntries(filePaths.map(path => [path, []]))
      const result = await ConvertToPDF(filePaths, validSheetSelections)
      pdfUrl = result
      addLog(`PDF変換が完了しました: ${result}`)
    } catch (error) {
      addLog(`PDF変換エラー: ${error}`)
    } finally {
      isConverting = false
    }
  }

  function addLog(message) {
    const timestamp = new Date().toLocaleTimeString()
    logs.push(`${timestamp}: ${message}`)
    logs = [...logs]
    
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

  function formatFileSize(bytes) {
    if (bytes < 1024) return bytes + ' B'
    if (bytes < 1024 * 1024) return Math.round(bytes / 1024) + ' KB'
    return Math.round(bytes / (1024 * 1024)) + ' MB'
  }

  // Resize handlers
  function startResizeLeftPanel(e) {
    isResizingLeftPanel = true
    e.preventDefault()
  }

  function startResizeRightPanel(e) {
    isResizingRightPanel = true
    e.preventDefault()
  }

  function startResizeFileTree(e) {
    isResizingFileTree = true
    e.preventDefault()
  }

  function startResizeSelectedFiles(e) {
    isResizingSelectedFiles = true
    e.preventDefault()
  }

  function handleMouseMove(e) {
    if (isResizingLeftPanel) {
      const containerRect = document.querySelector('.app-container').getBoundingClientRect()
      const newWidth = Math.max(250, Math.min(500, e.clientX - containerRect.left))
      leftPanelWidth = newWidth
    }
    
    if (isResizingRightPanel) {
      const rightPanel = document.querySelector('.right-panel')
      const rightRect = rightPanel.getBoundingClientRect()
      const relativeY = e.clientY - rightRect.top
      const newSplit = Math.max(30, Math.min(80, (relativeY / rightRect.height) * 100))
      rightPanelSplit = newSplit
    }

    if (isResizingFileTree) {
      const leftPanel = document.querySelector('.left-panel')
      const leftRect = leftPanel.getBoundingClientRect()
      const relativeY = e.clientY - leftRect.top - 10 // Account for header
      const panelHeight = leftRect.height - 10
      const newHeight = Math.max(20, Math.min(60, (relativeY / panelHeight) * 100))
      
      const remaining = 100 - newHeight
      const ratio = selectedFilesHeight / (selectedFilesHeight + sheetsHeight)
      
      fileTreeHeight = newHeight
      selectedFilesHeight = remaining * ratio
      sheetsHeight = remaining * (1 - ratio)
    }

    if (isResizingSelectedFiles) {
      const leftPanel = document.querySelector('.left-panel')
      const leftRect = leftPanel.getBoundingClientRect()
      const relativeY = e.clientY - leftRect.top - 10
      const panelHeight = leftRect.height - 10
      const treeBottom = (fileTreeHeight / 100) * panelHeight
      const availableHeight = panelHeight - treeBottom
      const newSelectedHeight = Math.max(15, Math.min(60, ((relativeY - treeBottom) / availableHeight) * 100))
      
      const totalRemaining = 100 - fileTreeHeight
      selectedFilesHeight = (newSelectedHeight / 100) * totalRemaining
      sheetsHeight = totalRemaining - selectedFilesHeight
    }
  }

  function handleMouseUp() {
    isResizingLeftPanel = false
    isResizingRightPanel = false
    isResizingFileTree = false
    isResizingSelectedFiles = false
  }
</script>

<main on:mousemove={handleMouseMove} on:mouseup={handleMouseUp}>
  <!-- Main Application Layout -->
  <div class="app-container">
    <!-- Left Panel -->
    <div class="left-panel" style="width: {leftPanelWidth}px;">
      <!-- File Tree -->
      <div class="panel-section file-tree-section" style="height: {fileTreeHeight}%;">
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

      <!-- Resize Handle for File Tree -->
      <div class="resize-handle horizontal" on:mousedown={startResizeFileTree}></div>

      <!-- Selected Files List -->
      <div class="panel-section selected-files-section" style="height: {selectedFilesHeight}%;">
        <div class="section-header-compact">
          <h3>選択ファイル</h3>
          <span class="count-badge">({selectedFiles.length})</span>
        </div>
        <div class="selected-files">
          {#each selectedFiles as file, index}
            <div class="selected-file-item" class:active={currentFile && currentFile.path === file.path}>
              <div class="file-info" on:click={() => selectFileFromList(file)} on:keydown={(e) => e.key === 'Enter' && selectFileFromList(file)} tabindex="0" role="button">
                <span class="file-icon">
                  {#if file.name.includes('.xls')}📊{:else if file.name.endsWith('.pdf')}📄{:else}📝{/if}
                </span>
                <span class="file-name">{file.name}</span>
              </div>
              <div class="file-controls">
                <button class="btn-small" on:click={() => moveFileUp(index)} disabled={index === 0}>↑</button>
                <button class="btn-small" on:click={() => moveFileDown(index)} disabled={index === selectedFiles.length - 1}>↓</button>
                <button class="btn-small btn-danger" on:click={() => removeFile(index)}>×</button>
              </div>
            </div>
          {/each}
        </div>
      </div>

      <!-- Resize Handle for Selected Files -->
      <div class="resize-handle horizontal" on:mousedown={startResizeSelectedFiles}></div>

      <!-- Excel Sheets -->
      <!-- Excel Sheets -->
      <div class="panel-section sheets-section" style="height: {sheetsHeight}%;">
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
        </div>
      </div>
    </div>

    <!-- Resize Handle for Left Panel -->
    <div class="resize-handle vertical" on:mousedown={startResizeLeftPanel}></div>

    <!-- Right Panel -->
    <div class="right-panel">
      <!-- PDF Viewer -->
      <div class="pdf-viewer-section" style="height: {effectiveRightPanelSplit}%;">
        <div class="section-header">
          <h3>PDFプレビュー</h3>
        </div>
        <div class="pdf-viewer-container">
          {#if pdfUrl}
            <embed src={pdfUrl} type="application/pdf" class="pdf-viewer" />
          {:else}
            <div class="pdf-placeholder">
              <div>
                <h3>PDFが生成されるとここに表示されます</h3>
                <p>左側でファイルを選択してPDFに変換してください</p>
              </div>
            </div>
          {/if}
        </div>
      </div>

      <!-- Resize Handle for Right Panel -->
      {#if isLogExpanded}
        <div class="resize-handle horizontal" on:mousedown={startResizeRightPanel}></div>
      {/if}

      <!-- Log Console -->
      <div class="log-section" style="height: {100 - effectiveRightPanelSplit}%;">
        <div class="section-header clickable" 
             on:click={() => isLogExpanded = !isLogExpanded} 
             on:keydown={(e) => e.key === 'Enter' && (isLogExpanded = !isLogExpanded)} 
             tabindex="0" 
             role="button">
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
    </div>
  </div>
</main>

<style>
  :global(*) {
    box-sizing: border-box;
  }

  :global(body) {
    margin: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    font-size: 14px;
    height: 100vh;
    overflow: hidden;
  }

  main {
    height: 100vh;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  /* Main application layout */
  .app-container {
    display: flex;
    flex: 1;
    border: 1px solid #ddd;
    border-radius: 8px;
    overflow: hidden;
    margin: 0.25rem;
  }

  /* Left panel */
  .left-panel {
    background: #f8f9fa;
    border-right: 1px solid #dee2e6;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    min-width: 250px;
    max-width: 500px;
    position: relative;
  }

  .panel-section {
    padding: 0.5rem;
    border-bottom: 1px solid #dee2e6;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    position: relative;
  }

  .panel-section h3 {
    margin: 0 0 0.25rem 0;
    font-size: 14px;
    font-weight: 600;
    color: #495057;
    flex-shrink: 0;
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

  /* File tree */
  .file-tree-section {
    min-height: 150px;
  }

  .file-tree {
    flex: 1;
    overflow-y: auto;
    border: 1px solid #dee2e6;
    border-radius: 4px;
    background: white;
    min-height: 0;
  }

  /* Selected files */
  .selected-files-section {
    min-height: 100px;
  }

  .selected-files {
    flex: 1;
    overflow-y: auto;
    border: 1px solid #dee2e6;
    border-radius: 4px;
    background: white;
    min-height: 0;
  }

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

  /* Sheets section */
  .sheets-section {
    min-height: 80px;
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

  .selected-file-item {
    display: flex;
    align-items: center;
    padding: 0.5rem;
    border-bottom: 1px solid #f8f9fa;
    gap: 0.5rem;
    background: white;
  }

  .selected-file-item:hover {
    background: #f8f9fa;
  }

  .selected-file-item.active {
    background: #e7f3ff;
    border-color: #007bff;
  }

  .file-info {
    display: flex;
    align-items: center;
    flex: 1;
    gap: 0.5rem;
    cursor: pointer;
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
  input[type="checkbox"] {
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

  /* Right panel */
  .right-panel {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
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

  /* PDF viewer */
  .pdf-viewer-section {
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .pdf-viewer-container {
    flex: 1;
    overflow: auto;
    background: #525659;
    position: relative;
    min-height: 0;
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

  /* Log section */
  .log-section {
    display: flex;
    flex-direction: column;
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
    min-height: 0;
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

  /* Responsive design */
  @media (max-width: 768px) {
    .app-container {
      flex-direction: column;
      height: auto;
    }
    
    .left-panel {
      width: 100% !important;
      max-width: none;
      max-height: 50vh;
    }
    
    .resize-handle {
      display: none;
    }
  }
</style>
