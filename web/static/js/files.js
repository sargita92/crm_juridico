/* F14 — Arquivos: drawer toggle + lightbox
 * Minimal JS: HTMX handles everything else.
 */
(function () {
    var drawer = null;
    function getDrawer() {
        if (!drawer) drawer = document.getElementById('file-drawer');
        return drawer;
    }

    window.openFileDrawer = function () {
        var d = getDrawer();
        if (d) {
            d.classList.add('open');
            d.setAttribute('aria-hidden', 'false');
        }
    };

    window.closeFileDrawer = function () {
        var d = getDrawer();
        if (d) {
            d.classList.remove('open');
            d.setAttribute('aria-hidden', 'true');
            d.innerHTML = '';
        }
    };

    window.openLightbox = function (src) {
        var lb = document.createElement('div');
        lb.className = 'files-lightbox';
        lb.setAttribute('role', 'dialog');
        lb.innerHTML =
            '<button type="button" class="close-lightbox" aria-label="Fechar">×</button>' +
            '<img src="' + src + '" alt="">';
        function close() { document.body.removeChild(lb); document.removeEventListener('keydown', onKey); }
        lb.addEventListener('click', function (e) {
            if (e.target === lb || e.target.classList.contains('close-lightbox')) close();
        });
        function onKey(e) { if (e.key === 'Escape') close(); }
        document.addEventListener('keydown', onKey);
        document.body.appendChild(lb);
    };

    document.addEventListener('keydown', function (e) {
        if (e.key === 'Escape') window.closeFileDrawer();
    });
})();
