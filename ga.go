// Package ga implements the genetic algorithm.
// It can handle negative fitness properly.
// Mutation probability is adaptive and does not need to be set.
package ga

import (
	"math"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

// Entity is an entity of GA model.
type Entity interface {
	// Fitness is the fitness of this entity.
	Fitness() float64
	// Mutate is the mutation operation.
	Mutate() Entity
	// Crossover is the crossover operation.
	Crossover(Entity, float64) Entity
}

// GA is a GA model.
type GA struct {
	n         int
	fitness   float64
	elite     Entity
	pm        float64
	base      float64
	fsum      float64
	rnd       *rand.Rand
	mutex     sync.Mutex
	fentities []float64
	entities  []Entity
	tentities []Entity
}

// NC is the number of concurrency, default to runtime.GOMAXPROCS.
var NC = runtime.GOMAXPROCS(0)

// New creates a GA model.
func New(n int, g func() Entity) *GA {
	m := &GA{
		n:         n,
		fitness:   math.Inf(-1),
		pm:        0.1,
		rnd:       rand.New(rand.NewSource(time.Now().Unix())),
		fentities: make([]float64, n),
		entities:  make([]Entity, n),
		tentities: make([]Entity, n),
	}
	m.do(func(c, i int) {
		m.entities[i] = g()
	})
	m.base = m.adjust()
	return m
}

// Fitness returns the fitness of current elite.
func (m *GA) Fitness() float64 {
	return m.fitness
}

// Elite returns the current elite.
func (m *GA) Elite() Entity {
	return m.elite
}

// Next gets the next generation of GA model, and returns the current elite and fitness.
func (m *GA) Next() (Entity, float64) {
	m.do(func(c, i int) {
		x, y, w := m.select2()
		z := x.Crossover(y, w)
		if m.rand() < m.pm {
			z = z.Mutate()
		}
		m.tentities[i] = z
	})
	m.entities, m.tentities = m.tentities, m.entities
	m.adjust()
	return m.elite, m.fitness
}

// Evolve runs the GA model until the elite k generations have not changed,
// or the max of iterations has been reached.
func (m *GA) Evolve(k int, max int) (Entity, float64, bool) {
	i, fitness := 0, m.fitness
	for j := 0; i < k && j < max; i, j = i+1, j+1 {
		_, f := m.Next()
		if fitness < f {
			i, fitness = 0, f
		}
	}
	return m.elite, fitness, i >= k
}

func (m *GA) adjust() float64 {
	sms, svs, mfs, mes := make([]float64, NC), make([]float64, NC), make([]float64, NC), make([]Entity, NC)
	for c := range mfs {
		mfs[c] = math.Inf(-1)
	}
	m.do(func(c, i int) {
		e := m.entities[i]
		f := e.Fitness()
		m.fentities[i] = f
		sms[c] += f
		svs[c] += f * f
		if mfs[c] < f {
			mfs[c], mes[c] = f, e
		}
	})
	if c, f := max(mfs); m.fitness < f {
		m.fitness, m.elite = f, mes[c]
	}

	mean, std := sum(sms)/float64(m.n), 1.0
	if v := sum(svs)/float64(m.n) - mean*mean; v > 0 {
		std = math.Sqrt(v)
	}
	if m.base > 0 {
		m.pm *= 0.2*math.Exp(-5*std/m.base) + 0.9
		if m.pm > 0.1 {
			m.pm = 0.1
		} else if m.pm < 0.0001 {
			m.pm = 0.0001
		}
	}

	fsums := make([]float64, NC)
	m.do(func(c, i int) {
		f := 1 / (1 + math.Exp((mean-m.fentities[i])/std))
		m.fentities[i] = f
		fsums[c] += f
	})
	m.fsum = sum(fsums)
	return std
}

func (m *GA) select2() (Entity, Entity, float64) {
	rx, ry := m.rand(), m.rand()
	if rx > ry {
		rx, ry = ry, rx
	}
	fz, d, isx := m.fsum*rx, m.fsum*(ry-rx), true
	x, y, wx, wy := m.entities[0], m.entities[m.n-1], 0.0, 0.0
	for i, f := range m.fentities {
		if fz <= f {
			if isx {
				x, wx, isx = m.entities[i], f, false
				fz = fz + d - f*ry
				continue
			} else {
				y, wy = m.entities[i], f
				break
			}
		}
		fz -= f
	}
	return x, y, wx / (wx + wy)
}

func (m *GA) rand() float64 {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.rnd.Float64()
}

func (m *GA) do(f func(c, i int)) {
	var wg sync.WaitGroup
	wg.Add(NC)
	for c := 0; c < NC; c++ {
		go func(c int) {
			defer wg.Done()
			for i := c; i < m.n; i += NC {
				f(c, i)
			}
		}(c)
	}
	wg.Wait()
}

func sum(xs []float64) float64 {
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s
}

func max(xs []float64) (int, float64) {
	k, m := 0, math.Inf(-1)
	for i, x := range xs {
		if x > m {
			k, m = i, x
		}
	}
	return k, m
}
