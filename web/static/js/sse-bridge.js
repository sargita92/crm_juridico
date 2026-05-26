// Bridge SSE por aba (F26): conecta ao SharedWorker (uma única conexão SSE por
// navegador) e entrega os eventos ao htmx desta aba.
//
// Só ativa em páginas tenant — detectadas pela presença de #toast-container (o
// sino de notificações, incluído em toda página tenant). Em páginas sem ele
// (ex.: admin) o bridge não faz nada.
//
// Sem SharedWorker (navegador antigo), o real-time degrada para os polls
// (every 5s/30s) que já existem nos templates — nada quebra.
(function () {
  "use strict";

  if (!document.getElementById("toast-container")) {
    return; // não é página tenant
  }
  if (typeof SharedWorker === "undefined") {
    return; // sem suporte: segue só com polling
  }

  var worker;
  try {
    worker = new SharedWorker("/static/js/sse-worker.js");
  } catch (err) {
    return; // worker indisponível: segue só com polling
  }

  worker.port.start();
  worker.port.onmessage = function (e) {
    var msg = e.data;
    if (!msg || !msg.event) {
      return;
    }

    if (msg.event === "notification") {
      // Fragmento HTML (toast + OOB do badge): deixa o htmx processar o swap/OOB.
      if (window.htmx && document.getElementById("toast-container")) {
        window.htmx.swap("#toast-container", msg.data || "", { swapStyle: "beforeend" });
      }
      return;
    }

    // Demais eventos (new-message, conversation-update, lead-*): dispara um evento
    // de DOM no body; os elementos htmx escutam via hx-trigger="<evento> from:body".
    document.body.dispatchEvent(new CustomEvent(msg.event));
  };
})();
