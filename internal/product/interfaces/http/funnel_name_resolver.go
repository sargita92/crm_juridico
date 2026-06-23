package http

// missingFunnelLabel é exibido quando um produto tem um vínculo (FunnelProduct)
// para um funil que não está mais visível para o tenant (deletado ou movido).
// Antes mostrávamos o UUID cru — o que confundia o usuário.
const missingFunnelLabel = "Funil indisponível"

// resolveFunnelName devolve o nome do funil pelo ID usando o mapa de funis
// disponíveis. Retorna missingFunnelLabel quando o ID não está no mapa,
// para evitar expor UUIDs na interface.
func resolveFunnelName(funnelID string, byID map[string]string) string {
	if name, ok := byID[funnelID]; ok && name != "" {
		return name
	}
	return missingFunnelLabel
}

// funnelNameIndex constrói o mapa ID→Name a partir da lista de funis.
func funnelNameIndex(funnels []FunnelInfo) map[string]string {
	idx := make(map[string]string, len(funnels))
	for _, f := range funnels {
		idx[f.ID] = f.Name
	}
	return idx
}
