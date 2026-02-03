// PDF.js viewer for modal
// Uses PDF.js from CDN (legacy build for non-module compatibility)

// Viewer state
let pdfDoc = null;
let pageNum = 1;
let scale = 1.0;
let rendering = false;

// Initialize PDF.js when available
function initPDFJS() {
    const pdfjsLib = window.pdfjsLib;
    if (!pdfjsLib) {
        console.error('PDF.js library not loaded');
        return false;
    }
    pdfjsLib.GlobalWorkerOptions.workerSrc =
        'https://unpkg.com/pdfjs-dist@4.10.38/build/pdf.worker.min.mjs';
    return true;
}

// Load PDF document
async function loadPDF(url) {
    if (!initPDFJS()) {
        document.getElementById('pdf-loading').innerHTML =
            '<p class="text-destructive">PDF viewer not available</p>';
        return;
    }

    try {
        const loadingTask = window.pdfjsLib.getDocument(url);
        pdfDoc = await loadingTask.promise;

        document.getElementById('page-count').textContent = pdfDoc.numPages;
        document.getElementById('page-num').textContent = '1';

        pageNum = 1;
        scale = 1.0;
        updateZoomDisplay();
        await renderPage(pageNum);
    } catch (error) {
        console.error('Error loading PDF:', error);
        document.getElementById('pdf-loading').innerHTML =
            '<p class="text-destructive">Failed to load PDF</p>';
    }
}

// Render a page
async function renderPage(num) {
    if (rendering || !pdfDoc) return;
    rendering = true;

    try {
        const page = await pdfDoc.getPage(num);
        const viewport = page.getViewport({ scale });

        const canvas = document.getElementById('pdf-canvas');
        const ctx = canvas.getContext('2d');

        // High-DPI support
        const outputScale = window.devicePixelRatio || 1;
        canvas.width = Math.floor(viewport.width * outputScale);
        canvas.height = Math.floor(viewport.height * outputScale);
        canvas.style.width = Math.floor(viewport.width) + 'px';
        canvas.style.height = Math.floor(viewport.height) + 'px';

        const transform = outputScale !== 1
            ? [outputScale, 0, 0, outputScale, 0, 0]
            : null;

        await page.render({
            canvasContext: ctx,
            transform: transform,
            viewport: viewport
        }).promise;

        document.getElementById('page-num').textContent = num;
        document.getElementById('pdf-loading').style.display = 'none';
        canvas.style.display = 'block';
    } catch (error) {
        console.error('Error rendering page:', error);
    } finally {
        rendering = false;
    }
}

// Navigation
function prevPage() {
    if (pageNum <= 1 || !pdfDoc) return;
    pageNum--;
    renderPage(pageNum);
}

function nextPage() {
    if (!pdfDoc || pageNum >= pdfDoc.numPages) return;
    pageNum++;
    renderPage(pageNum);
}

// Zoom
function zoomIn() {
    if (scale >= 3.0) return; // Max zoom 300%
    scale += 0.25;
    updateZoomDisplay();
    if (pdfDoc) renderPage(pageNum);
}

function zoomOut() {
    if (scale <= 0.5) return; // Min zoom 50%
    scale -= 0.25;
    updateZoomDisplay();
    if (pdfDoc) renderPage(pageNum);
}

function resetZoom() {
    scale = 1.0;
    updateZoomDisplay();
    if (pdfDoc) renderPage(pageNum);
}

function updateZoomDisplay() {
    const zoomDisplay = document.getElementById('zoom-level');
    if (zoomDisplay) {
        zoomDisplay.textContent = Math.round(scale * 100) + '%';
    }
}

// Fullscreen
function toggleFullscreen() {
    const container = document.getElementById('pdf-viewer-container');
    if (!container) return;

    if (!document.fullscreenElement) {
        container.requestFullscreen().catch(err => {
            console.error('Fullscreen error:', err);
        });
    } else {
        document.exitFullscreen();
    }
}

// Close modal and cleanup
function closePDFViewer() {
    const modal = document.getElementById('pdf-viewer-modal');
    if (modal) {
        modal.remove();
    }
    pdfDoc = null;
    pageNum = 1;
    scale = 1.0;
}

// Keyboard shortcuts
document.addEventListener('keydown', (e) => {
    const modal = document.getElementById('pdf-viewer-modal');
    if (!modal) return;

    // Don't handle if user is typing in an input
    if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;

    switch (e.key) {
        case 'Escape':
            closePDFViewer();
            break;
        case 'ArrowLeft':
            e.preventDefault();
            prevPage();
            break;
        case 'ArrowRight':
            e.preventDefault();
            nextPage();
            break;
        case '+':
        case '=':
            e.preventDefault();
            zoomIn();
            break;
        case '-':
            e.preventDefault();
            zoomOut();
            break;
        case '0':
            e.preventDefault();
            resetZoom();
            break;
    }
});

// Handle backdrop click
document.addEventListener('click', (e) => {
    if (e.target.id === 'pdf-viewer-backdrop') {
        closePDFViewer();
    }
});
