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

// --- Lista de seleção (modais com checklist) ---
// Atualiza um contador e habilita o botão de submit conforme o usuário marca
// checkboxes dentro de um container `data-tenants-list`. Usa delegação no body
// porque o container é re-renderizado pelo HTMX a cada busca, perdendo handlers
// presos diretamente nos filhos.
(function() {
    function updateChecklistState(scope) {
        var list = scope.querySelector("[data-tenants-list]");
        if (!list) return;
        var form = list.closest("form");
        if (!form) return;
        var counter = form.querySelector("[data-tenants-counter]");
        var submit = form.querySelector("[data-tenants-submit]");
        var count = list.querySelectorAll('input[type="checkbox"]:checked').length;
        if (counter) {
            counter.textContent = count === 0
                ? "Nenhum escritório selecionado"
                : (count === 1 ? "1 escritório selecionado" : count + " escritórios selecionados");
        }
        if (submit) {
            submit.disabled = count === 0;
        }
    }

    document.addEventListener("change", function(evt) {
        var list = evt.target.closest && evt.target.closest("[data-tenants-list]");
        if (!list) return;
        updateChecklistState(list.parentElement || document);
    });

    // Quando a lista é re-renderizada (busca, lazy-load), reaplica o estado
    // do contador — sem isso, o botão fica "habilitado" visualmente mesmo
    // após o reset do form.
    document.body.addEventListener("htmx:afterSwap", function(evt) {
        if (!evt.target || !evt.target.matches) return;
        if (evt.target.matches("[data-tenants-list]") ||
            (evt.target.querySelector && evt.target.querySelector("[data-tenants-list]"))) {
            updateChecklistState(evt.target.closest("form") || document);
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

