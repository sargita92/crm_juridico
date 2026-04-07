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

// Close modals on Escape key
document.addEventListener("keydown", function(e) {
    if (e.key === "Escape") {
        var modals = document.querySelectorAll(".modal-overlay");
        modals.forEach(function(m) { m.style.display = "none"; });
    }
});

// Close modals after successful HTMX swap
// Pages with "no-auto-close-modals" class manage their own modal lifecycle.
document.addEventListener("htmx:afterSwap", function() {
    if (document.body.classList.contains("no-auto-close-modals")) return;
    var modals = document.querySelectorAll(".modal-overlay");
    modals.forEach(function(m) { m.style.display = "none"; });
});
