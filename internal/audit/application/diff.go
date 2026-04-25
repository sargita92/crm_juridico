package application

import (
	"reflect"

	"github.com/sasrgita/crm-juridico/internal/audit/domain"
)

// diffIgnoredKeys lista chaves transientes que nunca devem aparecer num
// diff de auditoria (gerados automaticamente pelo banco/Gorm).
//
// Comparacao e case-insensitive (ver isDiffIgnored). Chaves de seguranca
// (`password`, `token`, etc.) sao filtradas via domain.IsForbiddenMetadataKey
// — fonte unica de verdade para a politica de redaction.
var diffIgnoredKeys = map[string]struct{}{
	"updated_at": {},
}

func isDiffIgnored(key string) bool {
	if domain.IsForbiddenMetadataKey(key) {
		return true
	}
	// Comparar lowercase para alinhar com a politica do dominio.
	for ignored := range diffIgnoredKeys {
		if equalFold(key, ignored) {
			return true
		}
	}
	return false
}

// equalFold inline (evita import de strings so para isso).
func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// BuildDiff produz um Metadata com o subset de campos que mudaram entre
// `before` e `after`.
//
// Formato de saida (alinhado ao QA-cenarios da F12):
//
//	{ "campo": {"antes": <valor antigo|nil>, "depois": <valor novo|nil>} }
//
// Comportamento:
//   - Profundidade 1: valores aninhados nao sao explorados.
//   - Ignora `updated_at` e qualquer chave em domain.IsForbiddenMetadataKey
//     (case-insensitive).
//   - Campo so em `before` -> `depois` = nil (interpretado como removido).
//   - Campo so em `after`  -> `antes` = nil (interpretado como adicionado).
//   - Sem diferencas: retorna nil (caller pode anexar diretamente em
//     `metadata.fields` sem checar tamanho).
func BuildDiff(before, after map[string]any) domain.Metadata {
	out := domain.Metadata{}

	// Conjunto de chaves: uniao de before + after.
	keys := make(map[string]struct{}, len(before)+len(after))
	for k := range before {
		keys[k] = struct{}{}
	}
	for k := range after {
		keys[k] = struct{}{}
	}

	for k := range keys {
		if isDiffIgnored(k) {
			continue
		}
		bVal, bOK := before[k]
		aVal, aOK := after[k]

		// Mesmo presenca em ambos + valores iguais -> sem mudanca.
		if bOK && aOK && reflect.DeepEqual(bVal, aVal) {
			continue
		}

		entry := map[string]any{}
		if bOK {
			entry["antes"] = bVal
		} else {
			entry["antes"] = nil
		}
		if aOK {
			entry["depois"] = aVal
		} else {
			entry["depois"] = nil
		}
		out[k] = entry
	}

	if len(out) == 0 {
		return nil
	}
	return out
}
