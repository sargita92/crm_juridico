package domain

import (
	"sync"
	"time"
)

// HolidayCalendar abstrai um calendário de feriados, usado pelo cron de
// pagamentos para calcular vencimentos prorrogando para o próximo dia útil.
type HolidayCalendar interface {
	IsHoliday(d time.Time) bool
	NextBusinessDay(d time.Time) time.Time
	AddBusinessDays(d time.Time, n int) time.Time
}

// BrazilHolidayCalendar implementa HolidayCalendar com os feriados nacionais
// brasileiros. Os feriados móveis (Carnaval, Sexta-feira Santa e Corpus
// Christi) são calculados a partir da Páscoa pelo algoritmo anônimo
// gregoriano de Meeus/Jones/Butcher. Os feriados são carregados sob demanda
// por ano e mantidos em cache.
//
// O calendário é seguro para uso concorrente: o cache por ano é protegido
// por sync.RWMutex, permitindo múltiplas leituras simultâneas (caminho
// quente, com cache hit) e serializando apenas a primeira escrita de cada
// ano (cache miss). Necessário porque o calendário é compartilhado entre o
// cron diário de pagamentos e chamadas concorrentes de handlers HTTP.
type BrazilHolidayCalendar struct {
	mu    sync.RWMutex
	cache map[int]map[string]bool
}

// NewBrazilHolidayCalendar cria um novo calendário de feriados brasileiros
// com cache vazio.
func NewBrazilHolidayCalendar() *BrazilHolidayCalendar {
	return &BrazilHolidayCalendar{cache: make(map[int]map[string]bool)}
}

// IsHoliday retorna true se d (truncado para a data, ignorando horário) for
// um feriado nacional brasileiro.
func (c *BrazilHolidayCalendar) IsHoliday(d time.Time) bool {
	day := truncateToDate(d)
	holidays := c.holidaysForYear(day.Year())
	return holidays[day.Format("2006-01-02")]
}

// NextBusinessDay retorna d (truncado para a data) se for dia útil
// (segunda a sexta e não feriado). Caso contrário, avança dia a dia até
// encontrar o próximo dia útil.
func (c *BrazilHolidayCalendar) NextBusinessDay(d time.Time) time.Time {
	day := truncateToDate(d)
	for !c.isBusinessDay(day) {
		day = day.AddDate(0, 0, 1)
	}
	return day
}

// AddBusinessDays normaliza d para o próximo dia útil (mesmo dia se já for
// útil) e então avança n dias úteis. Para n=0 retorna o próprio dia
// normalizado.
func (c *BrazilHolidayCalendar) AddBusinessDays(d time.Time, n int) time.Time {
	day := c.NextBusinessDay(d)
	for i := 0; i < n; i++ {
		day = day.AddDate(0, 0, 1)
		for !c.isBusinessDay(day) {
			day = day.AddDate(0, 0, 1)
		}
	}
	return day
}

// isBusinessDay retorna true se day for dia útil (não é sábado, domingo nem
// feriado).
func (c *BrazilHolidayCalendar) isBusinessDay(day time.Time) bool {
	switch day.Weekday() {
	case time.Saturday, time.Sunday:
		return false
	}
	return !c.IsHoliday(day)
}

// holidaysForYear devolve o conjunto de feriados (chaveado por YYYY-MM-DD)
// para o ano informado, carregando-o sob demanda.
//
// Usa o padrão double-check: tenta ler com lock compartilhado (RLock); se
// cache miss, adquire lock exclusivo (Lock) e revalida antes de escrever,
// evitando que duas goroutines que perderam o cache simultaneamente
// recomputem e sobrescrevam o conjunto de feriados.
func (c *BrazilHolidayCalendar) holidaysForYear(year int) map[string]bool {
	c.mu.RLock()
	cached, ok := c.cache[year]
	c.mu.RUnlock()
	if ok {
		return cached
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if cached, ok := c.cache[year]; ok {
		return cached
	}
	holidays := buildHolidays(year)
	c.cache[year] = holidays
	return holidays
}

// buildHolidays monta o conjunto de feriados nacionais brasileiros para um
// determinado ano. Inclui feriados fixos e os derivados da Páscoa.
func buildHolidays(year int) map[string]bool {
	holidays := make(map[string]bool, 12)

	fixed := []struct {
		month time.Month
		day   int
	}{
		{time.January, 1},    // Confraternização Universal
		{time.April, 21},     // Tiradentes
		{time.May, 1},        // Dia do Trabalho
		{time.September, 7},  // Independência
		{time.October, 12},   // Nossa Senhora Aparecida
		{time.November, 2},   // Finados
		{time.November, 15},  // Proclamação da República
		{time.December, 25},  // Natal
	}
	for _, f := range fixed {
		d := time.Date(year, f.month, f.day, 0, 0, 0, 0, time.UTC)
		holidays[d.Format("2006-01-02")] = true
	}

	easter := computeEaster(year)
	moveable := []time.Time{
		easter.AddDate(0, 0, -48), // Carnaval (segunda)
		easter.AddDate(0, 0, -47), // Carnaval (terça)
		easter.AddDate(0, 0, -2),  // Sexta-feira Santa
		easter.AddDate(0, 0, 60),  // Corpus Christi
	}
	for _, d := range moveable {
		holidays[d.Format("2006-01-02")] = true
	}

	return holidays
}

// computeEaster calcula a data do Domingo de Páscoa para o ano informado
// usando o algoritmo anônimo gregoriano (Meeus/Jones/Butcher).
func computeEaster(year int) time.Time {
	a := year % 19
	b := year / 100
	c := year % 100
	d := b / 4
	e := b % 4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i := c / 4
	k := c % 4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	day := ((h + l - 7*m + 114) % 31) + 1
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

// truncateToDate zera a parte de horário preservando a localização.
func truncateToDate(d time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
}
