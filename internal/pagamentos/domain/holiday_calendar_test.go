package domain_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

func TestNextBusinessDay_Weekday(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	wed := time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, wed, cal.NextBusinessDay(wed))
}

func TestNextBusinessDay_Saturday(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	sat := time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC)
	mon := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, mon, cal.NextBusinessDay(sat))
}

func TestNextBusinessDay_Sunday(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	sun := time.Date(2026, 5, 17, 0, 0, 0, 0, time.UTC)
	mon := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, mon, cal.NextBusinessDay(sun))
}

func TestNextBusinessDay_IndependenceDay(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	sep7 := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC) // segunda, feriado
	sep8 := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, sep8, cal.NextBusinessDay(sep7))
}

func TestNextBusinessDay_GoodFriday2026(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	// Pascoa 2026: 5 de abril (domingo). Sexta-feira santa: 3 de abril.
	goodFri := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	mon := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, mon, cal.NextBusinessDay(goodFri))
}

func TestNextBusinessDay_Christmas(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	// 25/12/2026 = sexta, feriado; volta para 28/12 (segunda)
	xmas := time.Date(2026, 12, 25, 0, 0, 0, 0, time.UTC)
	mon := time.Date(2026, 12, 28, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, mon, cal.NextBusinessDay(xmas))
}

func TestAddBusinessDays_Zero(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	mon := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, mon, cal.AddBusinessDays(mon, 0))
}

func TestAddBusinessDays_SkipsWeekend(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	start := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC) // sexta
	// +1 dia util = segunda 18/05 (pula 16 e 17)
	assert.Equal(t, time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC), cal.AddBusinessDays(start, 1))
	// +3 dias uteis
	assert.Equal(t, time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC), cal.AddBusinessDays(start, 3))
}

func TestAddBusinessDays_SkipsHoliday(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	// 20/04/2026 (segunda). +1 dia util deveria ser 21/04 (Tiradentes, terca)
	// mas Tiradentes eh feriado -> pula para 22/04 (quarta)
	start := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	expected := time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, cal.AddBusinessDays(start, 1))
}

func TestIsHoliday_FixedDates(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	fixed := []time.Time{
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),   // Confraternizacao
		time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC),  // Tiradentes
		time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),   // Trabalho
		time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC),   // Independencia
		time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC), // N.Sra.Aparecida
		time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC),  // Finados
		time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC), // Proclamacao
		time.Date(2026, 12, 25, 0, 0, 0, 0, time.UTC), // Natal
	}
	for _, d := range fixed {
		assert.Truef(t, cal.IsHoliday(d), "esperava que %s fosse feriado", d)
	}
}

func TestIsHoliday_EasterBased2026(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	// Pascoa 2026: domingo 5 de abril
	// Carnaval: segunda 16/02 e terca 17/02 (Pascoa - 48 e -47) -- corrigir
	// Sexta-feira santa: 3 de abril
	// Corpus Christi: quinta 4 de junho (60 dias apos Pascoa? quinta apos Corpus)
	// Carnaval eh o primeiro dia util apos a quarta de cinzas? Nao: carnaval = segunda e terca antes da quarta-feira de cinzas (que eh 46 dias antes da Pascoa).
	// Portanto: Carnaval segunda = Pascoa - 48 dias, Carnaval terca = Pascoa - 47 dias.
	// Sexta-feira santa = Pascoa - 2 dias.
	// Corpus Christi = Pascoa + 60 dias (quinta-feira).
	easter := time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC)
	assert.True(t, cal.IsHoliday(easter.AddDate(0, 0, -48))) // Carnaval segunda
	assert.True(t, cal.IsHoliday(easter.AddDate(0, 0, -47))) // Carnaval terca
	assert.True(t, cal.IsHoliday(easter.AddDate(0, 0, -2)))  // Sexta-feira santa
	assert.True(t, cal.IsHoliday(easter.AddDate(0, 0, 60)))  // Corpus Christi
}

func TestIsHoliday_NotHoliday(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	assert.False(t, cal.IsHoliday(time.Date(2026, 12, 23, 0, 0, 0, 0, time.UTC)))
	assert.False(t, cal.IsHoliday(time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC)))
}

// TestBrazilHolidayCalendar_ConcurrentAccess garante que o calendário é
// seguro para uso concorrente. O cache por ano é populado sob demanda e,
// sem sincronização, leituras + escritas simultâneas no mapa interno
// causariam pânico do runtime ("fatal error: concurrent map read and map
// write"). O teste lança 50 goroutines exercitando IsHoliday,
// NextBusinessDay e AddBusinessDays em múltiplos anos (2024-2028), de modo
// a forçar tanto cache miss (primeira chamada de cada ano) quanto cache
// hit. Deve ser executado com -race para flagrar qualquer regressão.
func TestBrazilHolidayCalendar_ConcurrentAccess(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()

	years := []int{2024, 2025, 2026, 2027, 2028}
	const goroutines = 50
	const iterationsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)
	start := make(chan struct{})

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			<-start // libera todas ao mesmo tempo, maximizando o stress

			for i := 0; i < iterationsPerGoroutine; i++ {
				year := years[(id+i)%len(years)]
				month := time.Month(((id + i) % 12) + 1)
				day := ((id+i)%27 + 1)
				d := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

				_ = cal.IsHoliday(d)
				_ = cal.NextBusinessDay(d)
				_ = cal.AddBusinessDays(d, (id+i)%5)
			}
		}(g)
	}

	close(start)
	wg.Wait()

	// Sanity check pós-stress: o cache continua íntegro e devolve resultados
	// determinísticos para feriados conhecidos em cada ano exercitado.
	for _, year := range years {
		newYear := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
		assert.Truef(t, cal.IsHoliday(newYear), "1 jan %d deve ser feriado após acesso concorrente", year)
	}
}

func TestEaster_KnownYears(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	// 2024: 31/03; 2025: 20/04; 2026: 05/04; 2027: 28/03; 2028: 16/04.
	// Verificar via Sexta-feira Santa que sempre eh 2 dias antes.
	cases := []struct {
		year   int
		easter time.Time
	}{
		{2024, time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)},
		{2025, time.Date(2025, 4, 20, 0, 0, 0, 0, time.UTC)},
		{2026, time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC)},
		{2027, time.Date(2027, 3, 28, 0, 0, 0, 0, time.UTC)},
		{2028, time.Date(2028, 4, 16, 0, 0, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		// Sexta-feira santa = Pascoa - 2 dias. Deve ser feriado.
		gf := c.easter.AddDate(0, 0, -2)
		assert.Truef(t, cal.IsHoliday(gf), "Good Friday %d (%s) should be holiday", c.year, gf)
	}
}
