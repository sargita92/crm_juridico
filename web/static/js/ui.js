/* =============================================================
   UI helpers — global HTMX progress bar + a11y polish
   Loads early so it's ready before HTMX fires its first request.
   ============================================================= */
(function () {
    'use strict';

    // --- Global progress bar (top sliver) ---
    var inflight = 0;
    function show() {
        inflight++;
        document.body.classList.add('htmx-loading');
    }
    function hide() {
        inflight = Math.max(0, inflight - 1);
        if (inflight === 0) document.body.classList.remove('htmx-loading');
    }

    document.addEventListener('htmx:beforeRequest', show);
    document.addEventListener('htmx:afterRequest', hide);
    document.addEventListener('htmx:responseError', hide);
    document.addEventListener('htmx:sendError', hide);
    document.addEventListener('htmx:timeout', hide);

    // --- Smooth content settling: ensure swapped content gets fade-in ---
    document.addEventListener('htmx:afterSwap', function (evt) {
        var t = evt.target;
        if (!t || t.classList.contains('no-anim')) return;
        // Trigger entrance only on substantial swaps (skip toasts/badges that already animate)
        if (t.matches('[data-anim], main, .admin-content, #tab-content')) {
            t.style.animation = 'none';
            // Force reflow then re-apply
            void t.offsetWidth;
            t.style.animation = '';
        }
    });
})();
