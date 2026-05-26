// SharedWorker do CRM (F26): mantém UMA única conexão SSE com /tenant/stream e
// repassa os eventos para todas as abas conectadas do mesmo navegador. Isso evita
// o estouro do limite de ~6 conexões por host do HTTP/1.1 quando há várias abas
// ou navegação rápida (cada page load reaproveita o worker em vez de abrir nova SSE).
//
// O navegador encerra o SharedWorker quando a última aba se desconecta, fechando
// a EventSource e liberando a conexão no servidor.

var ports = [];
var es = null;

// Eventos nomeados emitidos por /tenant/stream (ver internal/shared/events).
var EVENTS = [
  "notification",
  "new-message",
  "conversation-update",
  "lead-created",
  "lead-moved",
  "lead-responsible-assigned",
];

function broadcast(msg) {
  for (var i = ports.length - 1; i >= 0; i--) {
    try {
      ports[i].postMessage(msg);
    } catch (err) {
      ports.splice(i, 1); // porta morta: remove
    }
  }
}

function connect() {
  if (es) {
    return;
  }
  es = new EventSource("/tenant/stream"); // mesma origem: envia o cookie de sessão
  EVENTS.forEach(function (name) {
    es.addEventListener(name, function (evt) {
      broadcast({ event: name, data: evt.data });
    });
  });
  es.onerror = function () {
    // EventSource tenta reconectar sozinho; se fechou de vez, recria após um tempo.
    if (es && es.readyState === EventSource.CLOSED) {
      es = null;
      setTimeout(connect, 3000);
    }
  };
}

// eslint-disable-next-line no-unused-vars
self.onconnect = function (e) {
  var port = e.ports[0];
  ports.push(port);
  port.start();
  connect();
};
