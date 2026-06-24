// Sidebar toggle (mobile)
function toggleSidebar() {
    var sidebar = document.getElementById("admin-sidebar");
    var overlay = document.getElementById("sidebar-overlay");
    sidebar.classList.toggle("open");
    overlay.classList.toggle("open");
}

// Modal open/close
function openModal(id) {
    var modal = document.getElementById(id);
    if (modal) modal.style.display = "flex";
}

function closeModal(id) {
    var modal = document.getElementById(id);
    if (modal) modal.style.display = "none";
}

// Click no overlay (fora do modal-card) fecha o modal — mas só quando o
// mousedown ALSO acontece no overlay. Sem o pareamento, selecionar texto
// dentro do modal e soltar o mouse fora dispara click no overlay e fechava
// o modal acidentalmente (reportado em teste manual).
(function() {
    var pressedOnOverlay = null;
    document.addEventListener("mousedown", function(evt) {
        if (evt.target && evt.target.classList && evt.target.classList.contains("modal-overlay")) {
            pressedOnOverlay = evt.target;
        } else {
            pressedOnOverlay = null;
        }
    });
    document.addEventListener("click", function(evt) {
        var t = evt.target;
        if (!t || !t.classList || !t.classList.contains("modal-overlay")) return;
        if (pressedOnOverlay !== t) {
            // Press iniciou dentro do card — não fecha mesmo que o release tenha caído no overlay.
            return;
        }
        t.style.display = "none";
        pressedOnOverlay = null;
    });
})();

// Close modals and notification dropdown on Escape key
document.addEventListener("keydown", function(e) {
    if (e.key === "Escape") {
        var modals = document.querySelectorAll(".modal-overlay");
        modals.forEach(function(m) { m.style.display = "none"; });
        var dd = document.getElementById("notification-dropdown");
        if (dd) dd.style.display = "none";
    }
});

// Close modals after successful HTMX swap
// Pages with "no-auto-close-modals" class manage their own modal lifecycle.
document.addEventListener("htmx:afterSwap", function() {
    if (document.body.classList.contains("no-auto-close-modals")) return;
    var modals = document.querySelectorAll(".modal-overlay");
    modals.forEach(function(m) { m.style.display = "none"; });
});

// --- Modal "Gerenciar Escritórios" (lista única com diff) ---
// O modal renderiza TODOS os escritórios marcando os já associados via
// `data-initial="1"`. O usuário marca/desmarca livremente; calculamos um diff
// (added/removed) vs o estado inicial e exibimos algo como
// "2 a associar, 1 a desassociar". O submit só habilita quando há mudança.
// O backend (POST /tenants/sync) recebe a lista FINAL de IDs marcados e
// converge — não precisa receber explicitamente o diff. O listener fica em
// document.body porque a lista é re-renderizada pelo HTMX a cada busca.
(function() {
    function diffOf(list) {
        var checkboxes = list.querySelectorAll('input[type="checkbox"]');
        var added = 0, removed = 0;
        for (var i = 0; i < checkboxes.length; i++) {
            var cb = checkboxes[i];
            var initial = cb.getAttribute("data-initial") === "1";
            if (cb.checked && !initial) added++;
            else if (!cb.checked && initial) removed++;
        }
        return { added: added, removed: removed };
    }

    function formatDiff(d) {
        if (d.added === 0 && d.removed === 0) return "Nenhuma alteração";
        var parts = [];
        if (d.added > 0) parts.push(d.added + (d.added === 1 ? " a associar" : " a associar"));
        if (d.removed > 0) parts.push(d.removed + (d.removed === 1 ? " a desassociar" : " a desassociar"));
        return parts.join(" · ");
    }

    function updateState(scope) {
        var list = scope.querySelector("[data-tenants-list]");
        if (!list) return;
        var form = list.closest("form");
        if (!form) return;
        var diffEl = form.querySelector("[data-tenants-diff]");
        var submit = form.querySelector("[data-tenants-submit]");
        var d = diffOf(list);
        if (diffEl) diffEl.textContent = formatDiff(d);
        if (submit) submit.disabled = (d.added === 0 && d.removed === 0);
    }

    document.addEventListener("change", function(evt) {
        var list = evt.target.closest && evt.target.closest("[data-tenants-list]");
        if (!list) return;
        updateState(list.closest("form") || document);
    });

    // Re-render do HTMX (busca / abertura inicial) zera os handlers de filhos,
    // então reaplicamos o cálculo de diff/submit a partir do snapshot novo.
    document.body.addEventListener("htmx:afterSwap", function(evt) {
        if (!evt.target || !evt.target.matches) return;
        if (evt.target.matches("[data-tenants-list]") ||
            (evt.target.querySelector && evt.target.querySelector("[data-tenants-list]"))) {
            updateState(evt.target.closest("form") || document);
        }
    });
})();

// --- Admin toast (HTMX-friendly) ---
// Mostra um toast acionado por `HX-Trigger: {"adminToast": {"message": "...", "kind": "success"|"error"|"info"}}`.
// Reusa as classes .toast-container/.toast/.toast-success/.toast-error já presentes em main.css,
// criando o container on-demand para páginas admin que não embarcam o bell do tenant.
function showAdminToast(message, kind) {
    if (!message) return;
    var container = document.getElementById("admin-toast-container");
    if (!container) {
        container = document.createElement("div");
        container.id = "admin-toast-container";
        container.className = "toast-container";
        document.body.appendChild(container);
    }
    var el = document.createElement("div");
    el.className = "toast toast-" + (kind === "error" ? "error" : "success");
    el.setAttribute("role", "status");
    el.setAttribute("aria-live", "polite");
    el.textContent = message;
    container.appendChild(el);
    setTimeout(function() {
        if (el && el.parentNode) el.parentNode.removeChild(el);
    }, 4000);
}

document.body.addEventListener("adminToast", function(evt) {
    var d = evt.detail || {};
    showAdminToast(d.message, d.kind);
});

// --- Notification dropdown ---
function toggleNotificationDropdown(evt) {
    evt.stopPropagation();
    var dd = document.getElementById('notification-dropdown');
    if (!dd) return;
    dd.style.display = (dd.style.display === 'block') ? 'none' : 'block';
}

document.addEventListener('click', function(evt) {
    var dd = document.getElementById('notification-dropdown');
    var bell = document.getElementById('notification-bell');
    if (!dd || dd.style.display !== 'block') return;
    if (bell && bell.contains(evt.target)) return;
    dd.style.display = 'none';
});

