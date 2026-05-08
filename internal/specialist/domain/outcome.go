package domain

type Outcome string

const (
	OutcomeEmAndamento Outcome = "em_andamento"
	OutcomeAprovado    Outcome = "aprovado"
	OutcomeHumano      Outcome = "humano"
	OutcomeCrossSell   Outcome = "cross_sell"
	OutcomeReprovado   Outcome = "reprovado"
)

func CalculateOutcome(sc *ScoringConfig, score int) Outcome {
	if score >= sc.Threshold {
		return OutcomeAprovado
	}
	if sc.ThresholdHumanoMin > 0 && score >= sc.ThresholdHumanoMin {
		return OutcomeHumano
	}
	return OutcomeReprovado
}
