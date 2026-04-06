// Kanban drag-and-drop via Sortable.js
document.addEventListener('DOMContentLoaded', function() {
    initKanbanSortable();
});

// Re-init after HTMX swaps
document.addEventListener('htmx:afterSwap', function(e) {
    if (e.detail.target.id === 'kanban-content') {
        initKanbanSortable();
    }
});

function initKanbanSortable() {
    document.querySelectorAll('.kanban-column-body').forEach(function(el) {
        if (el._sortable) return; // already initialized
        el._sortable = new Sortable(el, {
            group: 'kanban',
            animation: 150,
            ghostClass: 'sortable-ghost',
            dragClass: 'sortable-drag',
            onEnd: function(evt) {
                var leadId = evt.item.dataset.leadId;
                var toColumnId = evt.to.dataset.columnId;
                if (!leadId || !toColumnId) return;

                // POST move via fetch, then refresh kanban
                var formData = new FormData();
                formData.append('column_id', toColumnId);

                fetch('/tenant/leads/' + leadId + '/move', {
                    method: 'POST',
                    body: formData
                }).then(function() {
                    // Trigger HTMX refresh of kanban
                    htmx.trigger('#kanban-content', 'refresh');
                });
            }
        });
    });
}

function closeLeadModal(event) {
    if (event.target === event.currentTarget) {
        document.getElementById('lead-modal').innerHTML = '';
    }
}

function closeFunnelModal(event) {
    if (event.target === event.currentTarget) {
        document.getElementById('funnel-modal').innerHTML = '';
    }
}
