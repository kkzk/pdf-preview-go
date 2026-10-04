import * as pdfjsLib from 'pdfjs-dist'
import workerUrl from 'pdfjs-dist/build/pdf.worker.min.mjs?url'
import 'pdfjs-dist/web/pdf_viewer.css'

pdfjsLib.GlobalWorkerOptions.workerSrc = workerUrl

// pdf_viewer.mjs reads the library from globalThis.pdfjsLib when it is
// evaluated, so it must be imported dynamically after this assignment
globalThis.pdfjsLib = pdfjsLib

let viewerModule

/**
 * Loads PDF.js and its viewer components.
 * @returns {Promise<{ pdfjsLib: typeof pdfjsLib } & typeof import('pdfjs-dist/web/pdf_viewer.mjs')>}
 */
export async function loadPdfjs() {
  viewerModule ??= await import('pdfjs-dist/web/pdf_viewer.mjs')
  return { pdfjsLib, ...viewerModule }
}
