<script>
  import {Greet, OpenDirectoryDialog, GetDirectoryContents, GetDirectoryTree, GetExcelSheets, ConvertToPDF, GetFileInfo, GetInitialDirectory} from '../wailsjs/go/main/App.js'
  import {onMount} from 'svelte'
  import TreeNode from './TreeNode.svelte'

  // Development test variables
  let resultText = "Please enter your name below 👇"
  let name

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
  let rightPanelSplit = 70 // percentage for PDF viewer
  let expandedFolders = new Set() // Track which folders are expanded

  // Initialize component
  onMount(async () => {
    try {
      // Get initial directory from command line argument
      const initialDir = await GetInitialDirectory()
      if (initialDir) {
        rootDirectory = initialDir
        await loadFileTree()
        addLog(`初期ディレクトリを設定しました: ${initialDir}`)
      }
    } catch (error) {
      addLog(`初期ディレクトリ取得エラー: ${error}`)
    }
  })

  function greet() {
    Greet(name).then(result => resultText = result)
  }

  async function selectRootDirectory() {
    try {
      const dir = await OpenDirectoryDialog()
      if (dir) {
        rootDirectory = dir
        await loadFileTree()
      }
    } catch (error) {
      addLog(`フォルダ選択エラー: ${error}`)
    }
  }

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
      const result = await ConvertToPDF(filePaths, sheetSelections)
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
    
    // Scroll to bottom of logs
    setTimeout(() => {
      const logContainer = document.querySelector('.log-container')
      if (logContainer) {
        logContainer.scrollTop = logContainer.scrollHeight
      }
    }, 100)
  }

  function formatFileSize(bytes) {
    if (bytes < 1024) return bytes + ' B'
    if (bytes < 1024 * 1024) return Math.round(bytes / 1024) + ' KB'
    return Math.round(bytes / (1024 * 1024)) + ' MB'
  }
</script>

<main>
  <!-- Development Test Area (collapsible) -->
  <details style="margin-bottom: 1rem;">
    <summary>Development Test Area</summary>
    <div class="dev-section">
      <div class="result">{resultText}</div>
      <div class="input-box">
        <input bind:value={name} class="input" placeholder="Enter your name" />
        <button class="btn" on:click={greet}>Greet</button>
      </div>
    </div>
  </details>

  <!-- Main Application Layout -->
  <div class="app-container">
    <!-- Left Panel -->
    <div class="left-panel" style="width: {leftPanelWidth}px;">
      <!-- Directory Selection -->
      <div class="panel-section">
        <h3>フォルダ選択</h3>
        <button class="btn-primary" on:click={selectRootDirectory}>
          📁 フォルダを選択
        </button>
        {#if rootDirectory}
          <div class="directory-info">
            <small>{rootDirectory}</small>
          </div>
        {/if}
      </div>

      <!-- File Tree -->
      <div class="panel-section file-tree-section">
        <h3>ファイル一覧</h3>
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

      <!-- Selected Files List -->
      <div class="panel-section selected-files-section">
        <h3>選択ファイル ({selectedFiles.length})</h3>
        <div class="selected-files">
          {#each selectedFiles as file, index}
            <div class="selected-file-item" class:active={currentFile && currentFile.path === file.path}>
              <div class="file-info" on:click={() => selectFileFromList(file)}>
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

      <!-- Excel Sheets -->
      <div class="panel-section sheets-section">
        <h3>シート選択</h3>
        {#if currentFile && excelSheets.length > 0}
          <div class="current-file-info">
            <strong>{currentFile.name}</strong>
          </div>
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
      <div class="panel-section">
        <button 
          class="btn-primary btn-large" 
          on:click={convertToPDF} 
          disabled={selectedFiles.length === 0 || isConverting}
        >
          {#if isConverting}変換中...{:else}📄 PDFに変換{/if}
        </button>
      </div>
    </div>

    <!-- Resize Handle -->
    <div class="resize-handle"></div>

    <!-- Right Panel -->
    <div class="right-panel">
      <!-- PDF Viewer -->
      <div class="pdf-viewer-section" style="height: {rightPanelSplit}%;">
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

      <!-- Log Console -->
      <div class="log-section" style="height: {100 - rightPanelSplit}%;">
        <div class="section-header">
          <h3>ログ</h3>
        </div>
        <div class="log-container">
          {#each logs as log}
            <div class="log-entry">{log}</div>
          {/each}
        </div>
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
  }

  /* Development section */
  .dev-section {
    padding: 1rem;
    background: #f8f9fa;
    border-radius: 4px;
  }

  .result {
    margin: 0.5rem 0;
    font-weight: 500;
  }

  .input-box {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .input {
    padding: 0.5rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    outline: none;
  }

  .btn {
    padding: 0.5rem 1rem;
    border: 1px solid #ddd;
    background: white;
    border-radius: 4px;
    cursor: pointer;
  }

  .btn:hover {
    background: #f8f9fa;
  }

  /* Main application layout */
  .app-container {
    display: flex;
    height: calc(100vh - 100px);
    border: 1px solid #ddd;
    border-radius: 8px;
    overflow: hidden;
  }

  /* Left panel */
  .left-panel {
    background: #f8f9fa;
    border-right: 1px solid #dee2e6;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    min-width: 250px;
    max-width: 400px;
  }

  .panel-section {
    padding: 1rem;
    border-bottom: 1px solid #dee2e6;
  }

  .panel-section h3 {
    margin: 0 0 0.5rem 0;
    font-size: 14px;
    font-weight: 600;
    color: #495057;
  }

  .directory-info {
    margin-top: 0.5rem;
    padding: 0.5rem;
    background: white;
    border-radius: 4px;
    border: 1px solid #dee2e6;
    word-break: break-all;
  }

  /* File tree */
  .file-tree-section {
    flex: 2;
    min-height: 0;
  }

  .file-tree {
    max-height: 300px;
    overflow-y: auto;
    border: 1px solid #dee2e6;
    border-radius: 4px;
    background: white;
  }

  /* Selected files */
  .selected-files-section {
    flex: 2;
    min-height: 0;
  }

  .selected-files {
    max-height: 200px;
    overflow-y: auto;
    border: 1px solid #dee2e6;
    border-radius: 4px;
    background: white;
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

  /* Sheets section */
  .sheets-section {
    flex: 1;
    min-height: 0;
  }

  .current-file-info {
    margin-bottom: 0.5rem;
    padding: 0.5rem;
    background: white;
    border-radius: 4px;
    border: 1px solid #dee2e6;
    font-size: 12px;
    color: #495057;
  }

  .sheets-list {
    max-height: 120px;
    overflow-y: auto;
    border: 1px solid #dee2e6;
    border-radius: 4px;
    background: white;
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

  /* Resize handle */
  .resize-handle {
    width: 4px;
    background: #dee2e6;
    cursor: col-resize;
  }

  .resize-handle:hover {
    background: #007bff;
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
    border-bottom: 1px solid #dee2e6;
  }

  .pdf-viewer-container {
    flex: 1;
    overflow: auto;
    background: #525659;
    position: relative;
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
    background: #2d3748;
    color: #e2e8f0;
    font-family: 'Consolas', 'Monaco', monospace;
    font-size: 11px;
    line-height: 1.4;
  }

  .log-entry {
    margin-bottom: 0.25rem;
    word-break: break-all;
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
