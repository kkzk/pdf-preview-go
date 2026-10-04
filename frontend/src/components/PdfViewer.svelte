<script>
  import { createEventDispatcher, onDestroy, onMount } from 'svelte'
  import { loadPdfjs } from '../lib/pdfjs.js'

  export let pdfUrl = ''
  export let hasUnsavedChanges = false

  const dispatch = createEventDispatcher()

  // Scale presets understood by PDFViewer.currentScaleValue
  const scaleOptions = [
    { value: 'page-width', label: 'ページ幅' },
    { value: 'page-fit', label: 'ページ全体' },
    { value: '0.5', label: '50%' },
    { value: '0.75', label: '75%' },
    { value: '1', label: '100%' },
    { value: '1.25', label: '125%' },
    { value: '1.5', label: '150%' },
    { value: '2', label: '200%' },
  ]
  const presetScales = ['page-width', 'page-fit', 'page-actual', 'auto']

  /** @type {HTMLDivElement} */
  let container
  let pdfjsLib
  let viewer
  let linkService
  let pdfDocument = null
  let loadingTask = null
  let loadSeq = 0
  let resizeObserver

  // Last viewed position ({ pageNumber, top, left } in PDF coordinates),
  // restored when the PDF is regenerated
  let lastLocation = null
  // Position to restore once the pages of a newly loaded PDF are laid out
  let pendingRestore = null

  let ready = false
  let loadError = ''
  let pageNumber = 1
  let pagesCount = 0
  let pageInput = '1'
  let scaleSelect = 'page-width'
  let scalePercent = 100

  onMount(async () => {
    const pdfjs = await loadPdfjs()
    pdfjsLib = pdfjs.pdfjsLib

    const eventBus = new pdfjs.EventBus()
    linkService = new pdfjs.PDFLinkService({ eventBus })
    // External links would navigate the whole app away
    linkService.externalLinkEnabled = false
    viewer = new pdfjs.PDFViewer({ container, eventBus, linkService })
    linkService.setViewer(viewer)

    eventBus.on('pagesinit', onPagesInit)
    eventBus.on('pagechanging', e => {
      pageNumber = e.pageNumber
      pageInput = String(e.pageNumber)
    })
    eventBus.on('scalechanging', e => {
      scalePercent = Math.round(e.scale * 100)
      // presetValue may also be a numeric value; use it only for named presets
      scaleSelect = presetScales.includes(e.presetValue) ? e.presetValue : findScaleOption(e.scale)
    })
    eventBus.on('updateviewarea', e => {
      lastLocation = e.location
    })

    container.addEventListener('wheel', onWheel, { passive: false })
    resizeObserver = new ResizeObserver(onResize)
    resizeObserver.observe(container)

    ready = true
  })

  onDestroy(() => {
    loadSeq++
    resizeObserver?.disconnect()
    container?.removeEventListener('wheel', onWheel)
    loadingTask?.destroy()
    destroyDocument(pdfDocument)
  })

  $: if (ready) loadPdf(pdfUrl)

  async function loadPdf(url) {
    const seq = ++loadSeq
    loadError = ''

    // Cancel a load that has been superseded (e.g. repeated conversions)
    loadingTask?.destroy()
    loadingTask = null

    if (!url) {
      showDocument(null)
      return
    }

    const task = pdfjsLib.getDocument({ url })
    loadingTask = task
    let doc
    try {
      doc = await task.promise
    } catch (error) {
      if (seq === loadSeq) {
        loadError = `PDFの読み込みに失敗しました: ${error?.message ?? error}`
      }
      return
    }
    if (seq !== loadSeq) {
      task.destroy() // A newer PDF has been requested
      return
    }
    loadingTask = null
    showDocument(doc)
  }

  // PDFDocumentProxy has no destroy() since PDF.js 6; destroy its loading task
  function destroyDocument(doc) {
    doc?.loadingTask.destroy()
  }

  function showDocument(doc) {
    // Remember where we were, unless this is the first PDF
    pendingRestore =
      pdfDocument && doc ? { location: lastLocation, scaleValue: viewer.currentScaleValue } : null

    const previous = pdfDocument
    pdfDocument = doc
    pagesCount = doc ? doc.numPages : 0
    linkService.setDocument(doc)
    viewer.setDocument(doc)
    destroyDocument(previous)
  }

  function onPagesInit() {
    const restore = pendingRestore
    pendingRestore = null

    if (!restore) {
      viewer.currentScaleValue = 'page-width'
      return
    }

    viewer.currentScaleValue = restore.scaleValue || 'page-width'
    const location = restore.location
    if (location) {
      viewer.scrollPageIntoView({
        pageNumber: Math.min(location.pageNumber, viewer.pagesCount),
        destArray: [null, { name: 'XYZ' }, location.left, location.top, null],
        allowNegativeOffset: true,
      })
    }
  }

  function findScaleOption(scale) {
    const option = scaleOptions.find(o => Math.abs(Number(o.value) - scale) < 0.001)
    return option ? option.value : 'custom'
  }

  function onScaleSelect() {
    if (scaleSelect !== 'custom') {
      viewer.currentScaleValue = scaleSelect
    }
  }

  function zoomIn() {
    viewer.increaseScale()
  }

  function zoomOut() {
    viewer.decreaseScale()
  }

  /** @param {WheelEvent} event */
  function onWheel(event) {
    if (!event.ctrlKey || !pdfDocument) {
      return
    }
    // Zoom the PDF instead of the whole app
    event.preventDefault()
    const options = { origin: [event.clientX, event.clientY] }
    if (event.deltaY < 0) {
      viewer.increaseScale(options)
    } else {
      viewer.decreaseScale(options)
    }
  }

  function onResize() {
    if (!pdfDocument) {
      return
    }
    // Recompute fit-to-width/page scales for the new size
    if (presetScales.includes(viewer.currentScaleValue)) {
      viewer.currentScaleValue = viewer.currentScaleValue
    } else {
      viewer.update()
    }
  }

  function goToPage(number) {
    if (!pdfDocument) {
      return
    }
    const page = Math.min(Math.max(1, Math.trunc(number) || 1), pagesCount)
    viewer.currentPageNumber = page
    pageInput = String(page)
  }

  function onPageInputKeydown(event) {
    if (event.key === 'Enter') {
      goToPage(Number(pageInput))
    }
  }

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
      <div class="pdf-toolbar">
        <div class="toolbar-group">
          <button
            class="btn-tool"
            on:click={() => goToPage(pageNumber - 1)}
            disabled={pageNumber <= 1}
            title="前のページ">◀</button
          >
          <input
            class="page-input"
            type="text"
            inputmode="numeric"
            bind:value={pageInput}
            on:keydown={onPageInputKeydown}
            on:blur={() => goToPage(Number(pageInput))}
            title="ページ番号"
          />
          <span class="page-count">/ {pagesCount}</span>
          <button
            class="btn-tool"
            on:click={() => goToPage(pageNumber + 1)}
            disabled={pageNumber >= pagesCount}
            title="次のページ">▶</button
          >
        </div>
        <div class="toolbar-group">
          <button class="btn-tool" on:click={zoomOut} title="縮小 (Ctrl+ホイール)">－</button>
          <select class="scale-select" bind:value={scaleSelect} on:change={onScaleSelect}>
            {#each scaleOptions as option (option.value)}
              <option value={option.value}>{option.label}</option>
            {/each}
            {#if scaleSelect === 'custom'}
              <option value="custom">{scalePercent}%</option>
            {/if}
          </select>
          <button class="btn-tool" on:click={zoomIn} title="拡大 (Ctrl+ホイール)">＋</button>
        </div>
        <button class="btn-save" on:click={saveCurrentPdf} title="PDFファイルを保存">
          💾 保存
        </button>
      </div>
    {/if}
  </div>
  <div class="pdf-viewer-container">
    <!-- PDFViewer requires an absolutely positioned scroll container -->
    <div class="viewer-scroll" bind:this={container}>
      <div class="pdfViewer"></div>
    </div>
    {#if !pdfUrl}
      <div class="pdf-placeholder">
        <div>
          <h3>PDFが生成されるとここに表示されます</h3>
          <p>左側でファイルを選択してPDFに変換してください。</p>
        </div>
      </div>
    {:else if loadError}
      <div class="pdf-placeholder">
        <p class="load-error">{loadError}</p>
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

  /* PDF header with toolbar and save button */
  .pdf-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
    flex-wrap: wrap;
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

  .pdf-toolbar {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-wrap: wrap;
  }

  .toolbar-group {
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }

  .btn-tool {
    min-width: 28px;
    height: 26px;
    padding: 0 0.4rem;
    border: 1px solid #ced4da;
    border-radius: 4px;
    background: white;
    color: #495057;
    font-size: 12px;
    cursor: pointer;
  }

  .btn-tool:hover:not(:disabled) {
    background: #e9ecef;
  }

  .btn-tool:disabled {
    color: #adb5bd;
    cursor: default;
  }

  .page-input {
    width: 3em;
    height: 24px;
    padding: 0 0.25rem;
    border: 1px solid #ced4da;
    border-radius: 4px;
    font-size: 12px;
    text-align: right;
  }

  .page-count {
    font-size: 12px;
    color: #495057;
    white-space: nowrap;
  }

  .scale-select {
    height: 26px;
    border: 1px solid #ced4da;
    border-radius: 4px;
    font-size: 12px;
    background: white;
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

  .viewer-scroll {
    position: absolute;
    inset: 0;
    overflow: auto;
    background: #525659;
  }

  .pdf-placeholder {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
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

  .load-error {
    color: #dc3545 !important;
  }
</style>
