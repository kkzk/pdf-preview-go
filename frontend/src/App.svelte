<script>
  import { onDestroy, onMount } from 'svelte'
  import {
    ConvertToPDF,
    GetAutoUpdateEnabled,
    GetDefaultSavePath,
    GetDirectoryContents,
    GetExcelSheets,
    GetFilesInfo,
    GetInitialDirectory,
    GetInitialFile,
    HasUnsavedChanges,
    LoadDirectorySessionCache,
    OpenFile,
    SaveDirectorySessionCache,
    SetAutoUpdateEnabled,
    SetWindowTitle,
    ShowSaveDialog,
  } from '../wailsjs/go/main/App.js'
  import { EventsOff, EventsOn } from '../wailsjs/runtime/runtime.js'
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

  // Session save interval reference
  let sessionSaveInterval
  $: effectiveRightPanelSplit = isLogExpanded ? rightPanelSplit : 95 // ログ折りたたみ時はPDF表示を95%に

  // Left panel section heights (percentages)
  let fileTreeHeight = 35
  let selectedFilesHeight = 20 // the sheet list gets the rest
  // sheetsHeight は削除 - CSS flexで自動調整

  // Resize states
  let isResizingLeftPanel = false
  let isResizingRightPanel = false
  let isResizingFileTree = false
  let isResizingSelectedFiles = false

  // Initialize component
  // Application exit confirmation handler

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
        }

        addLog(`作業ディレクトリを設定しました: ${initialDir}`)

        // Started with a file (e.g. from the Explorer context menu): select only
        // that file. Expanded folders and sheet selections come from the session.
        await selectInitialFile()
      }

      // Get auto-update setting
      autoUpdateEnabled = await GetAutoUpdateEnabled()

      // Initialize save status
      await updateSaveStatus()
    } catch (error) {
      addLog(`作業ディレクトリ取得エラー: ${error}`)
    }

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
        addLog(`作業フォルダを変更しました: ${newDir}`)
      }
    })

    // Listen for file change events
    EventsOn('file-changed', data => {
      const fileName = data.file.split('\\').pop() || data.file.split('/').pop()
      addLog(`ファイルが変更されました: ${fileName} - PDFを自動更新中...`)
      // The viewer reloads when the regenerated PDF URL arrives
      if (currentFile?.path === data.file) {
        refreshCurrentSheets()
      }
    })

    window.addEventListener('focus', handleWindowFocus)

    // Listen for conversion events
    EventsOn('conversion:error', data => {
      addLog(`自動更新エラー: ${data.message}`)
    })

    // Retries and files left out of the PDF
    EventsOn('conversion:warning', data => {
      addLog(`警告: ${data.message}`)
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
    window.removeEventListener('focus', handleWindowFocus)

    // Clear session save interval
    if (sessionSaveInterval) {
      clearInterval(sessionSaveInterval)
    }

    // Clean up event listeners
    EventsOff('directory-changed')
    EventsOff('file-changed')
    EventsOff('conversion:error')
    EventsOff('conversion:warning')
    EventsOff('conversion:progress')

    // Save session before component destroys
    if (rootDirectory) {
      saveCurrentDirectorySession().catch(error => {
        console.warn(`終了時セッション保存エラー: ${error}`)
      })
    }
  })

  async function selectInitialFile() {
    const filePath = await GetInitialFile()
    if (!filePath) {
      return
    }

    const [file] = await GetFilesInfo([filePath])
    if (!file) {
      addLog(`指定されたファイルが見つかりません: ${filePath}`)
      return
    }

    selectedFiles = [file]
    if (isExcelFile(file.name)) {
      await loadExcelSheets(file)
    } else {
      currentFile = file
      excelSheets = []
    }
    addLog(`ファイルを選択しました: ${file.name}`)
    debouncedSaveSession()
  }

  // Only the top level is loaded here; subfolders are loaded when expanded
  // (see loadFolderChildren), so large or cloud-backed folders open quickly
  async function loadFileTree() {
    try {
      fileTree = await GetDirectoryContents(rootDirectory)
      addLog(`フォルダを読み込みました: ${rootDirectory}`)
    } catch (error) {
      fileTree = []
      addLog(`フォルダ読み込みエラー: ${error}`)
    }
  }

  // Loads the contents of a folder in the tree, if not loaded yet
  async function loadFolderChildren(folderPath) {
    const node = findFileInTree(fileTree, folderPath)
    if (!node || !node.isDir || node.children || node.loading) {
      return
    }

    fileTree = updateTreeNode(fileTree, folderPath, n => ({ ...n, loading: true }))
    let children
    try {
      children = await GetDirectoryContents(folderPath)
    } catch (error) {
      children = []
      addLog(`フォルダ読み込みエラー: ${folderPath}: ${error}`)
    }
    fileTree = updateTreeNode(fileTree, folderPath, n => ({ ...n, loading: false, children }))
  }

  // Returns a copy of the tree in which the node at targetPath is replaced by
  // update(node); the nodes on the way are copied so that the change renders
  function updateTreeNode(tree, targetPath, update) {
    return tree.map(item => {
      if (item.path === targetPath) {
        return update(item)
      }
      if (item.children && isAncestorPath(item.path, targetPath)) {
        return { ...item, children: updateTreeNode(item.children, targetPath, update) }
      }
      return item
    })
  }

  function isAncestorPath(ancestor, path) {
    return path.startsWith(ancestor + '\\') || path.startsWith(ancestor + '/')
  }

  function toggleFolder(folderPath) {
    if (expandedFolders.has(folderPath)) {
      expandedFolders.delete(folderPath)
    } else {
      expandedFolders.add(folderPath)
      loadFolderChildren(folderPath)
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

  // Opens a file with its associated application (Excel, Word, ...)
  async function handleOpenFile(event) {
    const file = event.detail
    try {
      await OpenFile(file.path)
      addLog(`アプリで開きました: ${file.name}`)
    } catch (error) {
      addLog(`ファイルを開けませんでした: ${file.name}: ${error}`)
    }
  }

  function toggleFileSelection(file) {
    const index = selectedFiles.findIndex(f => f.path === file.path)
    const selecting = index < 0
    if (selecting) {
      selectedFiles.push({ ...file })
    } else {
      selectedFiles.splice(index, 1)
    }
    selectedFiles = [...selectedFiles]

    // Show a newly selected file so that its sheets can be chosen
    if (selecting) {
      if (isExcelFile(file.name)) {
        loadExcelSheets(file)
      } else {
        // Excel以外のファイルの場合、現在のファイルに設定してシート一覧をクリア
        currentFile = file
        excelSheets = []
      }
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

      // Initialize sheet selections if not exists (saved selections are
      // restored with the directory session)
      if (!sheetSelections[file.path]) {
        // Default: select all visible sheets
        sheetSelections[file.path] = excelSheets
          .filter(sheet => sheet.visible)
          .map(sheet => sheet.name)
      }

      reconcileSheetSelection(file, excelSheets)

      addLog(`Excelシートを読み込みました: ${file.name}`)
    } catch (error) {
      addLog(`Excelシート読み込みエラー: ${error}`)
    }
  }

  // Drop selected sheet names that no longer exist in the workbook (e.g. the
  // sheet was renamed or deleted). If none remain, select all visible sheets.
  // Returns whether the selection was changed.
  function reconcileSheetSelection(file, sheets) {
    const selected = sheetSelections[file.path]
    if (!selected || selected.length === 0) {
      return false
    }

    const existingNames = new Set(sheets.map(sheet => sheet.name))
    const kept = selected.filter(name => existingNames.has(name))
    if (kept.length === selected.length) {
      return false
    }

    const removed = selected.filter(name => !existingNames.has(name))
    if (kept.length > 0) {
      sheetSelections[file.path] = kept
    } else {
      sheetSelections[file.path] = sheets.filter(sheet => sheet.visible).map(sheet => sheet.name)
    }
    sheetSelections = { ...sheetSelections }
    addLog(`存在しないシートを選択から外しました: ${file.name} [${removed.join(', ')}]`)

    debouncedSaveSession()
    return true
  }

  // Reloads the sheet list of the current file, e.g. after sheets were added,
  // removed or renamed in Excel. Sheets added while all visible sheets were
  // selected are selected too. If the selection changes, the PDF shown is
  // regenerated, since auto-update converted it with the old selection.
  async function refreshCurrentSheets() {
    const file = currentFile
    if (!file || !isExcelFile(file.name)) {
      return
    }

    let sheets
    try {
      sheets = await GetExcelSheets(file.path)
    } catch (error) {
      return // e.g. the file is being saved; the next change will refresh it
    }
    if (currentFile?.path !== file.path || sameSheets(excelSheets, sheets)) {
      return
    }

    const oldSheets = excelSheets
    excelSheets = sheets
    let selectionChanged = false

    const added = sheets.filter(s => s.visible && !oldSheets.some(old => old.name === s.name))
    const selected = sheetSelections[file.path]
    if (added.length > 0 && selected && selected.length > 0) {
      const addedNames = added.map(s => s.name)
      const allWereSelected = oldSheets
        .filter(s => s.visible)
        .every(s => selected.includes(s.name))
      if (allWereSelected) {
        sheetSelections[file.path] = [...selected, ...addedNames]
        sheetSelections = { ...sheetSelections }
        selectionChanged = true
        addLog(`追加されたシートを選択に加えました: ${file.name} [${addedNames.join(', ')}]`)
      } else {
        addLog(`シートが追加されました（未選択）: ${file.name} [${addedNames.join(', ')}]`)
      }
    }

    if (reconcileSheetSelection(file, sheets)) {
      selectionChanged = true
    }

    if (selectionChanged) {
      debouncedSaveSession()
      if (pdfUrl && selectedFiles.some(f => f.path === file.path)) {
        convertToPDF()
      }
    }
  }

  function sameSheets(a, b) {
    return (
      a.length === b.length &&
      a.every((sheet, i) => sheet.name === b[i].name && sheet.visible === b[i].visible)
    )
  }

  // Sheets may have been changed in Excel while the app was in the
  // background. .xls is skipped, as reading it starts Excel.
  function handleWindowFocus() {
    if (currentFile && !currentFile.name.toLowerCase().endsWith('.xls')) {
      refreshCurrentSheets()
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

    // Debounced session save (includes sheet selections)
    debouncedSaveSession()
  }

  // Event handlers for SelectedFilesPanel
  function handleSelectFile(event) {
    const file = event.detail
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

      // Restore expanded folders, loading their contents parents first
      if (sessionCache.expandedFolders && sessionCache.expandedFolders.length > 0) {
        expandedFolders = new Set(sessionCache.expandedFolders)
        const folders = [...sessionCache.expandedFolders].sort((a, b) => a.length - b.length)
        for (const folder of folders) {
          await loadFolderChildren(folder)
        }
      }

      // Restore selected files (they may be in folders that are not loaded)
      if (sessionCache.selectedFiles && sessionCache.selectedFiles.length > 0) {
        selectedFiles = await GetFilesInfo(sessionCache.selectedFiles)
      }

      // Restore current file
      if (sessionCache.currentFile) {
        const [file] = await GetFilesInfo([sessionCache.currentFile])
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
      if (currentFile && excelSheets.length > 0) {
        reconcileSheetSelection(currentFile, excelSheets)
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
    const wasResizing =
      isResizingLeftPanel || isResizingRightPanel || isResizingFileTree || isResizingSelectedFiles
    isResizingLeftPanel = false
    isResizingRightPanel = false
    isResizingFileTree = false
    isResizingSelectedFiles = false
    if (wasResizing) {
      saveLayout()
    }
  }

  // Panel sizes are remembered per user in localStorage
  const layoutStorageKey = 'pdf-preview-go:layout'

  function saveLayout() {
    try {
      localStorage.setItem(
        layoutStorageKey,
        JSON.stringify({ leftPanelWidth, rightPanelSplit, fileTreeHeight, selectedFilesHeight })
      )
    } catch (error) {
      // Storage may be unavailable; the default layout is used next time
    }
  }

  function loadLayout() {
    try {
      const layout = JSON.parse(localStorage.getItem(layoutStorageKey) || '{}')
      const isNumber = value => typeof value === 'number' && Number.isFinite(value)
      if (isNumber(layout.leftPanelWidth)) leftPanelWidth = layout.leftPanelWidth
      if (isNumber(layout.rightPanelSplit)) rightPanelSplit = layout.rightPanelSplit
      if (isNumber(layout.fileTreeHeight)) fileTreeHeight = layout.fileTreeHeight
      if (isNumber(layout.selectedFilesHeight)) selectedFilesHeight = layout.selectedFilesHeight
    } catch (error) {
      // Use the default layout
    }
  }

  loadLayout()
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
          currentPath={currentFile?.path ?? ''}
          on:toggle-folder={handleToggleFolder}
          on:toggle-selection={handleToggleSelection}
          on:select-file={handleSelectFile}
          on:open-file={handleOpenFile}
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
        <PdfViewer {pdfUrl} {hasUnsavedChanges} on:save-pdf={saveCurrentPdf} />
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
