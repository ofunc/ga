// Package ga implements the genetic algorithm.
package ga

import (
	"math"
	"math/rand"
	"time"
)

// Entity is an entity of GA model.
type Entity interface {
	Fitness() float64
	Mutate() Entity
	Crossover(Entity) Entity
}

// GA is a GA model.
type GA struct {
	n        int
	elite    Entity
	felite   float64
	entities []Entity

	rnd       *rand.Rand
	fsum      float64
	mean      float64
	std       float64
	std0      float64
	fentities []float64
	tentities []Entity
}

// New creates a GA model.
func New(n int, g func() Entity) *GA {
	m := &GA{
		n:         n,
		felite:    math.Inf(-1),
		entities:  make([]Entity, n),
		rnd:       rand.New(rand.NewSource(time.Now().Unix())),
		fentities: make([]float64, n),
		tentities: make([]Entity, n),
	}
	for i := range m.entities {
		m.entities[i] = g()
	}
	m.fitness()
	m.std0 = m.std
	return m
}

// Elite returns the current elite.
func (m *GA) Elite() Entity {
	return m.elite
}

// Felite returns the fitness of current elite.
func (m *GA) Felite() float64 {
	return m.felite
}

// Next gets the next generation of GA model, and returns the current elite.
func (m *GA) Next() Entity {
	pm := math.Exp(-10 * m.std / m.std0)
	for i := range m.tentities {
		x, y := m.select2()
		z := x.Crossover(y)
		if m.rnd.Float64() < pm {
			z = z.Mutate()
		}
		m.tentities[i] = z
	}
	m.entities, m.tentities = m.tentities, m.entities
	m.fitness()
	return m.elite
}

// Evolve runs the GA model until the elite k generations have not changed,
// or the max of iterations has been reached.
func (m *GA) Evolve(k int, max int) (Entity, bool) {
	i, elite := 0, m.elite
	for j := 0; i < k && j < max; i, j = i+1, j+1 {
		x := m.Next()
		if x != elite {
			i, elite = 0, x
		}
	}
	return elite, i >= k
}

func (m *GA) fitness() {
	mean, std2 := 0.0, 0.0
	for i, x := range m.entities {
		f := x.Fitness()
		m.fentities[i] = f

		k1 := 1 / float64(i+1)
		k2 := 1 - k1
		mean = k1*f + k2*mean
		std2 = k1*f*f + k2*std2

		if m.felite < f {
			m.elite, m.felite = x, f
		}
	}
	m.mean = mean
	std2 -= mean * mean
	if std2 > 0 {
		m.std = math.Sqrt(std2)
	} else {
		m.std = 1
	}
	m.sigmoid()
}

func (m *GA) sigmoid() {
	m.fsum = 0
	for i, f := range m.fentities {
		f = 1 / (1 + math.Exp((m.mean-f)/m.std))
		m.fentities[i] = f
		m.fsum += f
	}
}

func (m *GA) select2() (Entity, Entity) {
	r1, r2 := m.rnd.Float64(), m.rnd.Float64()
	if r1 > r2 {
		r1, r2 = r2, r1
	}
	f0, d := m.fsum*r1, m.fsum*(r2-r1)
	x, y := m.entities[0], m.entities[m.n-1]
	for i, f := range m.fentities {
		if f0 <= f {
			if x == m.entities[0] {
				x = m.entities[i]
				f0 = f0 + d - f*r2
				continue
			} else {
				y = m.entities[i]
				break
			}
		}
		f0 -= f
	}
	return x, y
}
