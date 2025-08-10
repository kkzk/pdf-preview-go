<script>
  import { onDestroy, onMount } from 'svelte'
  import {
    ConvertToPDF,
    GetAutoUpdateEnabled,
    GetDefaultSavePath,
    GetDirectoryContents,
    GetDirectoryTree,
    GetExcelSheets,
    GetInitialDirectory,
    HasUnsavedChanges,
    LoadDirectorySessionCache,
    LoadSheetSelectionsForDirectory,
    SaveDirectorySessionCache,
    SaveSheetSelectionsForDirectory,
    SetAutoUpdateEnabled,
    SetWindowTitle,
    ShowSaveDialog,
  } from '../wailsjs/go/main/App.js'
  import { EventsOff, EventsOn, Quit } from '../wailsjs/runtime/runtime.js'
  import FileTreePanel from './components/FileTreePanel.svelte'
  import LogPanel from './components/LogPanel.svelte'
  import PdfViewer from './components/PdfViewer.svelte'
  import SelectedFilesPanel from './components/SelectedFilesPanel.svelte'
  import SheetsPanel from './components/SheetsPanel.svelte'

  // Helper function to check if file is Excel
  function isExcelFile(filename) {
    const ext = filename.toLowerCase()
    return ext.endsWith('.xlsx') || ext.endsWith('.xlsm') || ext.endsWith('.xls')
  }

  // Main application state
  let rootDirectory = ''
  let fileTree = []
  let selectedFiles = []
  let currentFile = null
  let excelSheets = []
  let sheetSelections = /** @type {Record<string, string[]>} */ ({})
  let pdfUrl = ''
  let logs = []
  let isConverting = false
  let autoUpdateEnabled = true
  let hasUnsavedChanges = false
  let defaultSavePath = ''

  // UI state
  let leftPanelWidth = 300
  let rightPanelSplit = 70 // percentage for PDF viewer when log is expanded
  let expandedFolders = new Set() // Track which folders are expanded
  let isLogExpanded = false // Track log section state
  let pdfViewerKey = 0 // Force PDF viewer reload

  // Session save interval reference
  let sessionSaveInterval
  $: effectiveRightPanelSplit = isLogExpanded ? rightPanelSplit : 95 // ログ折りたたみ時はPDF表示を95%に

  // Force PDF viewer reload when URL changes
  $: if (pdfUrl) {
    pdfViewerKey++
  }

  // Left panel section heights (percentages)
  let fileTreeHeight = 40
  let selectedFilesHeight = 35
  // sheetsHeight は削除 - CSS flexで自動調整

  // Resize states
  let isResizingLeftPanel = false
  let isResizingRightPanel = false
  let isResizingFileTree = false
  let isResizingSelectedFiles = false

  // Initialize component
  // Application exit confirmation handler
  let handleBeforeUnload

  onMount(async () => {
    try {
      // Get initial directory from command line argument
      const initialDir = await GetInitialDirectory()
      if (initialDir) {
        rootDirectory = initialDir
        await loadFileTree()
        await SetWindowTitle(initialDir)

        // Load saved session state for this directory
        try {
          await loadDirectorySession(initialDir)
        } catch (error) {
          addLog(`セッション状態の読み込みでエラー: ${error}`)

          // Fallback to loading only sheet selections
          try {
            const savedSelections = await LoadSheetSelectionsForDirectory()
            if (savedSelections && Object.keys(savedSelections).length > 0) {
              sheetSelections = savedSelections
              addLog(
                `保存されたシート選択を読み込みました (${Object.keys(savedSelections).length}ファイル)`
              )
            }
          } catch (fallbackError) {
            addLog(`シート選択の読み込みでエラー: ${fallbackError}`)
          }
        }

        addLog(`作業ディレクトリを設定しました: ${initialDir}`)
      }

      // Get auto-update setting
      autoUpdateEnabled = await GetAutoUpdateEnabled()

      // Initialize save status
      await updateSaveStatus()
    } catch (error) {
      addLog(`作業ディレクトリ取得エラー: ${error}`)
    }

    // Setup application exit confirmation
    handleBeforeUnload = async event => {
      try {
        const hasUnsaved = await HasUnsavedChanges()
        if (hasUnsaved && pdfUrl) {
          event.preventDefault()
          event.returnValue = '' // Required for Chrome

          // Show confirmation dialog
          const shouldSave = confirm(
            '未保存のPDFがあります。保存してからアプリケーションを終了しますか？\n\n「OK」: PDFを保存してから終了\n「キャンセル」: 保存せずに終了\n「×」: 終了をキャンセル'
          )

          if (shouldSave) {
            try {
              await ShowSaveDialog()
              addLog('PDFファイルを保存しました')
              // Allow normal exit after saving
              window.removeEventListener('beforeunload', handleBeforeUnload)
              Quit()
            } catch (saveError) {
              addLog(`保存エラー: ${saveError}`)
              // Don't quit if save failed
              return false
            }
          } else {
            // User chose not to save, allow exit
            window.removeEventListener('beforeunload', handleBeforeUnload)
            Quit()
          }
          return false
        }
      } catch (error) {
        addLog(`終了処理エラー: ${error}`)
      }
    }

    // Add beforeunload event listener
    window.addEventListener('beforeunload', handleBeforeUnload)

    // Listen for directory change events from menu
    EventsOn('directory-changed', async newDir => {
      // Save current session before changing directory
      if (rootDirectory) {
        await saveCurrentDirectorySession()
      }

      rootDirectory = newDir
      expandedFolders.clear()
      expandedFolders = new Set()

      // Reset current state
      selectedFiles = []
      currentFile = null
      excelSheets = []
      sheetSelections = {}
      pdfUrl = ''

      await loadFileTree()
      await SetWindowTitle(newDir)

      // Load saved session state for new directory
      try {
        await loadDirectorySession(newDir)
        addLog(`作業フォルダを変更し、前回の状態を復元しました: ${newDir}`)
      } catch (error) {
        addLog(`セッション状態の読み込みでエラー: ${error}`)

        // Fallback to loading only sheet selections
        try {
          const savedSelections = await LoadSheetSelectionsForDirectory()
          if (savedSelections && Object.keys(savedSelections).length > 0) {
            sheetSelections = savedSelections
            addLog(
              `新しいディレクトリのシート選択を読み込みました (${Object.keys(savedSelections).length}ファイル)`
            )
          }
        } catch (fallbackError) {
          addLog(`シート選択の読み込みでエラー: ${fallbackError}`)
        }

        addLog(`作業フォルダを変更しました: ${newDir}`)
      }
    })

    // Listen for file change events
    EventsOn('file-changed', data => {
      const fileName = data.file.split('\\').pop() || data.file.split('/').pop()
      addLog(`ファイルが変更されました: ${fileName} - PDFを自動更新中...`)
      // Force PDF viewer reload when file changes
      pdfViewerKey++
    })

    // Listen for conversion events
    EventsOn('conversion:error', data => {
      addLog(`自動更新エラー: ${data.message}`)
    })

    // Listen for conversion progress events
    EventsOn('conversion:progress', async status => {
      if (status.status === 'completed' && status.outputPath) {
        pdfUrl = status.outputPath
        addLog(`PDFが更新されました`)
        // Update save status after PDF generation
        await updateSaveStatus()
      }
    })

    // Auto-save session every 30 seconds
    sessionSaveInterval = setInterval(() => {
      if (rootDirectory) {
        saveCurrentDirectorySession().catch(error => {
          console.warn(`自動セッション保存エラー: ${error}`)
        })
      }
    }, 30000) // 30 seconds
  })

  onDestroy(() => {
    // Clear session save interval
    if (sessionSaveInterval) {
      clearInterval(sessionSaveInterval)
    }

    // Clean up event listeners
    EventsOff('directory-changed')
    EventsOff('file-changed')
    EventsOff('conversion:error')
    EventsOff('conversion:progress')

    // Clean up beforeunload event listener
    window.removeEventListener('beforeunload', handleBeforeUnload)

    // Save session before component destroys
    if (rootDirectory) {
      saveCurrentDirectorySession().catch(error => {
        console.warn(`終了時セッション保存エラー: ${error}`)
      })
    }
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

    // Debounced session save
    debouncedSaveSession()
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
      selectedFiles.push({ ...file })
    }
    selectedFiles = [...selectedFiles]

    // If it's an Excel file, load its sheets
    if (isExcelFile(file.name)) {
      loadExcelSheets(file)
    } else {
      // Excel以外のファイルの場合、現在のファイルに設定してシート一覧をクリア
      currentFile = file
      excelSheets = []
    }

    addLog(`ファイル選択更新: ${file.name}`)

    // Debounced session save
    debouncedSaveSession()
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
        // Check if we have saved selections for this file
        const savedSelections = await LoadSheetSelectionsForDirectory()
        if (savedSelections && savedSelections[file.path]) {
          // Use saved selections
          sheetSelections[file.path] = savedSelections[file.path]
          addLog(
            `保存されたシート選択を復元: ${file.name} [${sheetSelections[file.path].join(', ')}]`
          )
        } else {
          // Default: select all visible sheets
          sheetSelections[file.path] = excelSheets
            .filter(sheet => sheet.visible)
            .map(sheet => sheet.name)
        }
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
      addLog(`シート選択解除: ${sheetName}`)
    } else {
      sheetSelections[filePath].push(sheetName)
      addLog(`シート選択追加: ${sheetName}`)
    }

    sheetSelections = { ...sheetSelections }
    addLog(`${currentFile.name}の選択シート: [${sheetSelections[filePath].join(', ')}]`)

    // Save sheet selections automatically
    saveSheetSelections()

    // Debounced session save
    debouncedSaveSession()
  }

  // Save sheet selections to cache
  async function saveSheetSelections() {
    try {
      await SaveSheetSelectionsForDirectory(sheetSelections)
    } catch (error) {
      // Silently handle save errors - not critical for user experience
      console.warn('Failed to save sheet selections:', error)
    }
  }

  function selectFileFromList(file) {
    currentFile = file

    if (isExcelFile(file.name)) {
      loadExcelSheets(file)
    } else {
      // Excel以外のファイルの場合、シート一覧をクリア
      excelSheets = []
    }

    // Debounced session save
    debouncedSaveSession()
  }

  // Event handlers for SelectedFilesPanel
  function handleSelectFile(event) {
    selectFileFromList(event.detail)
  }

  function handleMoveFile(event) {
    const { from, to } = event.detail
    const temp = selectedFiles[from]
    selectedFiles[from] = selectedFiles[to]
    selectedFiles[to] = temp
    selectedFiles = [...selectedFiles]
    addLog('ファイル順序を変更しました')

    // Debounced session save
    debouncedSaveSession()
  }

  function handleRemoveFile(event) {
    const index = event.detail
    const removed = selectedFiles.splice(index, 1)[0]
    selectedFiles = [...selectedFiles]
    addLog(`ファイルを削除しました: ${removed.name}`)

    // Debounced session save
    debouncedSaveSession()
  }

  // Event handlers for SheetsPanel
  function handleToggleSheet(event) {
    toggleSheetSelection(event.detail)
  }

  function handleConvertPDF() {
    convertToPDF()
  }

  function handleToggleAutoUpdate() {
    toggleAutoUpdate()
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

      // Build valid sheet selections - if no sheets selected, use all visible sheets
      /** @type {Record<string, string[]>} */
      const validSheetSelections = {}
      for (const filePath of filePaths) {
        if (sheetSelections[filePath] && sheetSelections[filePath].length > 0) {
          validSheetSelections[filePath] = sheetSelections[filePath]
          addLog(`${filePath}: 選択されたシート [${sheetSelections[filePath].join(', ')}]`)
        } else {
          // If no sheets are selected, don't add to validSheetSelections
          // This will cause the converter to export all sheets
          validSheetSelections[filePath] = []
          addLog(`${filePath}: 全シートを出力`)
        }
      }

      addLog(`最終的なシート選択情報: ${JSON.stringify(validSheetSelections)}`)
      const result = await ConvertToPDF(filePaths, validSheetSelections)
      pdfUrl = result
      addLog(`PDF変換が完了しました: ${result}`)

      // Update save status after conversion
      await updateSaveStatus()
    } catch (error) {
      addLog(`PDF変換エラー: ${error}`)
    } finally {
      isConverting = false
    }
  }

  // Save related functions
  async function saveCurrentPdf() {
    if (!pdfUrl) {
      addLog('保存するPDFがありません')
      return
    }

    try {
      addLog('PDFの保存を開始します...')
      await ShowSaveDialog()

      // If we reach here, save was successful
      await updateSaveStatus()
      addLog('PDFファイルを保存しました')
    } catch (error) {
      // Handle different types of errors
      const errorStr = error ? error.toString() : ''

      if (errorStr.includes('user_cancelled')) {
        addLog('保存がキャンセルされました')
      } else if (errorStr.includes('cancelled') || errorStr.includes('cancel')) {
        addLog('保存がキャンセルされました')
      } else if (error) {
        addLog(`保存エラー: ${errorStr}`)
        console.error('Save error:', error)
      } else {
        addLog('保存がキャンセルされました')
      }

      // Update status even after error
      await updateSaveStatus()
    }
  }

  async function updateSaveStatus() {
    try {
      hasUnsavedChanges = await HasUnsavedChanges()
      defaultSavePath = await GetDefaultSavePath()
    } catch (error) {
      console.error('Failed to update save status:', error)
    }
  }

  // Session management functions
  async function saveCurrentDirectorySession() {
    if (!rootDirectory) {
      return
    }

    try {
      const selectedFilePaths = selectedFiles.map(f => f.path)
      const expandedFolderPaths = Array.from(expandedFolders)
      const currentFilePath = currentFile ? currentFile.path : ''

      await SaveDirectorySessionCache(
        rootDirectory,
        selectedFilePaths,
        expandedFolderPaths,
        currentFilePath,
        sheetSelections
      )
    } catch (error) {
      console.warn(`セッション保存エラー: ${error}`)
    }
  }

  async function loadDirectorySession(dirPath) {
    if (!dirPath) {
      return
    }

    try {
      const sessionCache = await LoadDirectorySessionCache(dirPath)
      if (!sessionCache) {
        // No session data found
        return
      }

      // Restore expanded folders
      if (sessionCache.expandedFolders && sessionCache.expandedFolders.length > 0) {
        expandedFolders = new Set(sessionCache.expandedFolders)
      }

      // Restore selected files
      if (sessionCache.selectedFiles && sessionCache.selectedFiles.length > 0) {
        selectedFiles = []
        for (const filePath of sessionCache.selectedFiles) {
          const file = findFileInTree(fileTree, filePath)
          if (file) {
            selectedFiles.push(file)
          }
        }
      }

      // Restore current file
      if (sessionCache.currentFile) {
        const file = findFileInTree(fileTree, sessionCache.currentFile)
        if (file) {
          currentFile = file
          // Load excel sheets for current file if it's an Excel file
          if (isExcelFile(file.name)) {
            try {
              excelSheets = await GetExcelSheets(file.path)
            } catch (error) {
              addLog(`シート情報取得エラー: ${error}`)
            }
          } else {
            // Excel以外の場合はシート一覧をクリア
            excelSheets = []
          }
        }
      }

      // Restore sheet selections
      if (sessionCache.sheetSelections) {
        sheetSelections = sessionCache.sheetSelections
      }

      const restoredItems = []
      if (sessionCache.selectedFiles?.length > 0) {
        restoredItems.push(`選択ファイル: ${sessionCache.selectedFiles.length}件`)
      }
      if (sessionCache.expandedFolders?.length > 0) {
        restoredItems.push(`展開フォルダ: ${sessionCache.expandedFolders.length}件`)
      }
      if (sessionCache.currentFile) {
        restoredItems.push(`現在のファイル`)
      }
      if (Object.keys(sessionCache.sheetSelections || {}).length > 0) {
        restoredItems.push(
          `シート選択: ${Object.keys(sessionCache.sheetSelections).length}ファイル`
        )
      }

      if (restoredItems.length > 0) {
        addLog(`前回の状態を復元しました (${restoredItems.join(', ')})`)
      }
    } catch (error) {
      throw new Error(`セッション復元エラー: ${error}`)
    }
  }

  function findFileInTree(tree, targetPath) {
    for (const item of tree) {
      if (item.path === targetPath) {
        return item
      }
      if (item.children) {
        const found = findFileInTree(item.children, targetPath)
        if (found) {
          return found
        }
      }
    }
    return null
  }

  // Debounced session save to avoid too frequent saves
  let saveSessionTimeout
  function debouncedSaveSession() {
    if (saveSessionTimeout) {
      clearTimeout(saveSessionTimeout)
    }

    saveSessionTimeout = setTimeout(() => {
      if (rootDirectory) {
        saveCurrentDirectorySession().catch(error => {
          console.warn(`遅延セッション保存エラー: ${error}`)
        })
      }
    }, 2000) // 2 seconds debounce
  }

  function addLog(message) {
    const timestamp = new Date().toLocaleTimeString()
    logs.push(`${timestamp}: ${message}`)
    logs = [...logs]
  }

  async function toggleAutoUpdate() {
    autoUpdateEnabled = !autoUpdateEnabled
    try {
      await SetAutoUpdateEnabled(autoUpdateEnabled)
      addLog(`自動更新を${autoUpdateEnabled ? '有効' : '無効'}にしました`)
    } catch (error) {
      addLog(`自動更新設定エラー: ${error}`)
      // Revert on error
      autoUpdateEnabled = !autoUpdateEnabled
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
      const ratio = selectedFilesHeight / (selectedFilesHeight + 35) // sheetsHeight基準値を35に固定

      fileTreeHeight = newHeight
      selectedFilesHeight = remaining * ratio
      // sheetsHeight は自動調整されるため削除
    }

    if (isResizingSelectedFiles) {
      const leftPanel = document.querySelector('.left-panel')
      const leftRect = leftPanel.getBoundingClientRect()
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
      // sheetsHeight は自動調整されるため削除
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
        <FileTreePanel
          {fileTree}
          {selectedFiles}
          {expandedFolders}
          on:toggle-folder={handleToggleFolder}
          on:toggle-selection={handleToggleSelection}
        />
      </div>

      <!-- Resize Handle for File Tree -->
      <div class="resize-handle horizontal" on:mousedown={startResizeFileTree}></div>

      <!-- Selected Files List -->
      <div class="panel-section selected-files-section" style="height: {selectedFilesHeight}%;">
        <SelectedFilesPanel
          {selectedFiles}
          {currentFile}
          on:select-file={handleSelectFile}
          on:move-file={handleMoveFile}
          on:remove-file={handleRemoveFile}
        />
      </div>

      <!-- Resize Handle for Selected Files -->
      <div class="resize-handle horizontal" on:mousedown={startResizeSelectedFiles}></div>

      <!-- Excel Sheets -->
      <!-- Excel Sheets -->
      <div class="panel-section sheets-section">
        <SheetsPanel
          {currentFile}
          {excelSheets}
          {sheetSelections}
          {selectedFiles}
          {isConverting}
          {autoUpdateEnabled}
          on:toggle-sheet={handleToggleSheet}
          on:convert-pdf={handleConvertPDF}
          on:toggle-auto-update={handleToggleAutoUpdate}
        />
      </div>
    </div>

    <!-- Resize Handle for Left Panel -->
    <div class="resize-handle vertical" on:mousedown={startResizeLeftPanel}></div>

    <!-- Right Panel -->
    <div class="right-panel">
      <!-- PDF Viewer -->
      <div class="pdf-viewer-container">
        <PdfViewer {pdfUrl} {pdfViewerKey} {hasUnsavedChanges} on:save-pdf={saveCurrentPdf} />
      </div>

      <!-- Resize Handle for Right Panel -->
      {#if isLogExpanded}
        <div class="resize-handle horizontal" on:mousedown={startResizeRightPanel}></div>
      {/if}

      <!-- Log Console -->
      <LogPanel {logs} bind:isLogExpanded {effectiveRightPanelSplit} />
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

  /* File tree */
  .file-tree-section {
    min-height: 150px;
  }

  /* Selected files */
  .selected-files-section {
    min-height: 100px;
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
    flex: 1; /* 残りのスペースを自動的に占有 */
  }

  /* Right panel */
  .right-panel {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  /* PDF Viewer container - takes all available space */
  .pdf-viewer-container {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    background-color: red;
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
