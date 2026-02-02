/**
 * Upload functionality with drag-and-drop and per-file progress tracking.
 * Uses XMLHttpRequest for upload progress (Fetch API doesn't support upload progress).
 */
(function() {
    'use strict';

    // DOM elements
    const dropOverlay = document.getElementById('drop-overlay');
    const uploadArea = document.getElementById('upload-area');
    const fileInput = document.getElementById('file-input');
    const selectFilesBtn = document.getElementById('select-files-btn');
    const progressContainer = document.getElementById('upload-progress');
    const resultsContainer = document.getElementById('upload-results');
    const toastContainer = document.getElementById('toast-container');

    // Drag counter to handle child element events (prevents overlay flicker)
    let dragCounter = 0;

    // Track upload state
    let activeUploads = 0;
    let uploadResults = { success: 0, duplicate: 0, failed: 0 };

    /**
     * Show the drop overlay
     */
    function showOverlay() {
        dropOverlay.classList.remove('hidden');
    }

    /**
     * Hide the drop overlay
     */
    function hideOverlay() {
        dropOverlay.classList.add('hidden');
    }

    /**
     * Show a toast notification
     * @param {string} message - Toast message
     * @param {boolean} isError - Whether this is an error toast
     */
    function showToast(message, isError = false) {
        const toast = document.createElement('div');
        toast.className = `px-4 py-3 rounded-lg shadow-lg flex items-center gap-3 animate-in slide-in-from-right duration-300 ${isError ? 'bg-red-600' : 'bg-green-600'} text-white`;

        const icon = isError
            ? '<svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path></svg>'
            : '<svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path></svg>';

        toast.innerHTML = `${icon}<span>${message}</span>`;
        toastContainer.appendChild(toast);

        // Auto-dismiss after 4 seconds
        setTimeout(() => {
            toast.remove();
        }, 4000);
    }

    /**
     * Create progress entry for a file
     * @param {string} filename - Name of the file
     * @param {number} index - Index for unique ID
     * @returns {HTMLElement} The progress entry element
     */
    function createProgressEntry(filename, index) {
        const entry = document.createElement('div');
        entry.id = `progress-${index}`;
        entry.className = 'border border-border rounded-lg p-4';
        entry.innerHTML = `
            <div class="flex items-center justify-between mb-2">
                <span class="font-medium truncate mr-4">${escapeHtml(filename)}</span>
                <span id="progress-text-${index}" class="text-sm text-muted-foreground">0%</span>
            </div>
            <div class="h-2 bg-muted rounded-full overflow-hidden">
                <div id="progress-bar-${index}" class="h-full bg-primary rounded-full transition-all duration-150" style="width: 0%"></div>
            </div>
        `;
        progressContainer.appendChild(entry);
        return entry;
    }

    /**
     * Update progress for a file
     * @param {number} index - Index of the progress entry
     * @param {number} percent - Percentage complete (0-100)
     */
    function updateProgress(index, percent) {
        const bar = document.getElementById(`progress-bar-${index}`);
        const text = document.getElementById(`progress-text-${index}`);
        if (bar) bar.style.width = `${percent}%`;
        if (text) text.textContent = `${percent}%`;
    }

    /**
     * Mark upload as complete with status
     * @param {number} index - Index of the progress entry
     * @param {boolean} success - Whether upload succeeded
     * @param {boolean} isDuplicate - Whether file was a duplicate
     * @param {string} error - Error message if failed
     */
    function markComplete(index, success, isDuplicate, error) {
        const entry = document.getElementById(`progress-${index}`);
        if (!entry) return;

        const bar = document.getElementById(`progress-bar-${index}`);
        const text = document.getElementById(`progress-text-${index}`);

        if (success && !isDuplicate) {
            if (bar) bar.className = 'h-full bg-green-500 rounded-full';
            if (text) text.textContent = 'Done';
            entry.className = 'border border-green-500 rounded-lg p-4';
        } else if (isDuplicate) {
            if (bar) bar.className = 'h-full bg-yellow-500 rounded-full';
            if (text) text.textContent = 'Duplicate';
            entry.className = 'border border-yellow-500 rounded-lg p-4';
        } else {
            if (bar) bar.className = 'h-full bg-red-500 rounded-full';
            if (text) text.textContent = error || 'Failed';
            entry.className = 'border border-red-500 rounded-lg p-4';
        }
    }

    /**
     * Escape HTML to prevent XSS
     * @param {string} str - String to escape
     * @returns {string} Escaped string
     */
    function escapeHtml(str) {
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }

    /**
     * Upload a single file with progress tracking
     * @param {File} file - File to upload
     * @param {number} index - Index for progress tracking
     * @returns {Promise<Object>} Upload result
     */
    function uploadFile(file, index) {
        return new Promise((resolve) => {
            const xhr = new XMLHttpRequest();
            const formData = new FormData();
            formData.append('file', file);

            // Track upload progress
            xhr.upload.onprogress = (e) => {
                if (e.lengthComputable) {
                    const percent = Math.round((e.loaded / e.total) * 100);
                    updateProgress(index, percent);
                }
            };

            // Handle completion
            xhr.onload = () => {
                activeUploads--;

                if (xhr.status >= 200 && xhr.status < 300) {
                    try {
                        const result = JSON.parse(xhr.responseText);
                        const success = result.success !== false;
                        const isDuplicate = result.is_duplicate === true;

                        markComplete(index, success, isDuplicate, result.error);

                        if (success && !isDuplicate) {
                            uploadResults.success++;
                        } else if (isDuplicate) {
                            uploadResults.duplicate++;
                        } else {
                            uploadResults.failed++;
                        }

                        resolve(result);
                    } catch (e) {
                        markComplete(index, false, false, 'Invalid response');
                        uploadResults.failed++;
                        resolve({ success: false, error: 'Invalid response' });
                    }
                } else {
                    let error = 'Upload failed';
                    try {
                        const result = JSON.parse(xhr.responseText);
                        error = result.error || error;
                    } catch (e) {
                        // Use default error
                    }
                    markComplete(index, false, false, error);
                    uploadResults.failed++;
                    resolve({ success: false, error: error });
                }

                // Show summary toast when all uploads complete
                checkAllComplete();
            };

            // Handle errors
            xhr.onerror = () => {
                activeUploads--;
                markComplete(index, false, false, 'Network error');
                uploadResults.failed++;
                resolve({ success: false, error: 'Network error' });
                checkAllComplete();
            };

            // Send the request
            xhr.open('POST', '/api/upload');
            xhr.send(formData);
            activeUploads++;
        });
    }

    /**
     * Check if all uploads are complete and show summary toast
     */
    function checkAllComplete() {
        if (activeUploads === 0) {
            const { success, duplicate, failed } = uploadResults;
            const total = success + duplicate + failed;

            if (total > 0) {
                let message = '';
                const parts = [];

                if (success > 0) {
                    parts.push(`${success} uploaded`);
                }
                if (duplicate > 0) {
                    parts.push(`${duplicate} duplicate${duplicate > 1 ? 's' : ''}`);
                }
                if (failed > 0) {
                    parts.push(`${failed} failed`);
                }

                message = parts.join(', ');
                showToast(message, failed > 0 && success === 0);

                // Reset counters for next batch
                uploadResults = { success: 0, duplicate: 0, failed: 0 };
            }
        }
    }

    /**
     * Filter and upload files
     * @param {FileList|File[]} files - Files to upload
     */
    function uploadFiles(files) {
        // Filter for PDF files only
        const pdfFiles = Array.from(files).filter(file => {
            const isPDF = file.type === 'application/pdf' || file.name.toLowerCase().endsWith('.pdf');
            if (!isPDF) {
                showToast(`Skipped "${file.name}" - not a PDF file`, true);
            }
            return isPDF;
        });

        if (pdfFiles.length === 0) {
            return;
        }

        // Clear previous progress entries
        progressContainer.innerHTML = '';

        // Create progress entries for each file
        pdfFiles.forEach((file, index) => {
            createProgressEntry(file.name, index);
        });

        // Upload all files in parallel
        pdfFiles.forEach((file, index) => {
            uploadFile(file, index);
        });
    }

    // === Event Listeners ===

    // Full-page drag events (use document for full-page drop zone)
    document.addEventListener('dragenter', (e) => {
        // Only react to file drags
        if (e.dataTransfer && e.dataTransfer.types && e.dataTransfer.types.includes('Files')) {
            e.preventDefault();
            dragCounter++;
            showOverlay();
        }
    });

    document.addEventListener('dragleave', (e) => {
        e.preventDefault();
        dragCounter--;
        if (dragCounter === 0) {
            hideOverlay();
        }
    });

    document.addEventListener('dragover', (e) => {
        if (e.dataTransfer && e.dataTransfer.types && e.dataTransfer.types.includes('Files')) {
            e.preventDefault();
            e.dataTransfer.dropEffect = 'copy';
        }
    });

    document.addEventListener('drop', (e) => {
        e.preventDefault();
        dragCounter = 0;
        hideOverlay();

        if (e.dataTransfer && e.dataTransfer.files && e.dataTransfer.files.length > 0) {
            uploadFiles(e.dataTransfer.files);
        }
    });

    // Click on upload area to trigger file input
    uploadArea.addEventListener('click', (e) => {
        // Don't trigger if clicking the button directly
        if (e.target !== selectFilesBtn) {
            fileInput.click();
        }
    });

    // Select files button
    selectFilesBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        fileInput.click();
    });

    // File input change
    fileInput.addEventListener('change', () => {
        if (fileInput.files && fileInput.files.length > 0) {
            uploadFiles(fileInput.files);
            fileInput.value = ''; // Reset for same file selection
        }
    });

})();
