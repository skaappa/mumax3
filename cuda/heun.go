package cuda

import (
	"code.google.com/p/mx3/data"
	"code.google.com/p/mx3/util"
	"log"
	"math"
)

// Adaptive heun solver.
// TODO: now only for magnetization (because it normalizes), post-step hook?
type Heun struct {
	solverCommon
	y, dy0      *data.Slice
	torqueFn    func(m *data.Slice, time float64) *data.Slice // updates dy
	normalizeFn func(m *data.Slice)                           // normalizes y
	Fixdt       bool                                          // fixed time step?
}

func NewHeun(y *data.Slice, torqueFn func(*data.Slice, float64) *data.Slice, normalizeFn func(m *data.Slice), dt, multiplier float64) *Heun {
	util.Argument(dt > 0 && multiplier > 0)
	dy0 := NewSlice(3, y.Mesh())
	return &Heun{newSolverCommon(dt, multiplier), y, dy0, torqueFn, normalizeFn, false}
}

// Take one time step
func (e *Heun) Step() {
	y, dy0 := e.y, e.dy0
	dt := float32(e.dt_si * e.dt_mul) // could check here if it is in float32 ranges
	util.Assert(dt > 0)

	// stage 1
	dy := e.torqueFn(y, e.Time)
	Madd2(y, y, dy, 1, dt) // y = y + dt * dy
	data.Copy(dy0, dy)

	// stage 2
	dy = e.torqueFn(y, e.Time)
	{
		err := 0.0
		if !e.Fixdt {
			err = MaxVecDiff(dy0, dy) * float64(dt)
			solverCheckErr(err)
		}

		if err < e.Maxerr || e.dt_si <= e.Mindt { // mindt check to avoid infinite loop
			Madd3(y, y, dy, dy0, 1, 0.5*dt, -0.5*dt)
			e.normalizeFn(y)
			e.Time += e.dt_si
			e.NSteps++
			if !e.Fixdt {
				e.adaptDt(math.Pow(e.Maxerr/err, 1./2.))
			}
		} else { // undo.
			util.Assert(!e.Fixdt)
			Madd2(y, y, dy0, 1, -dt)
			e.undone++
			e.adaptDt(math.Pow(e.Maxerr/err, 1./3.))
		}
		util.Dashf("step: % 8d (%6d) t: % 12es Δt: % 12es ε:% 12e", e.NSteps, e.undone, e.Time, e.dt_si, err)
	}
}

// Run for a duration in seconds
func (e *Heun) Advance(seconds float64) {
	log.Println("heun solver:", seconds, "s")
	stop := e.Time + seconds
	for e.Time < stop {
		e.Step()
	}
	util.DashExit()
}

// Run for a number of steps
func (e *Heun) Steps(steps int) {
	log.Println("heun solver:", steps, "steps")
	for s := 0; s < steps; s++ {
		e.Step()
	}
	util.DashExit()
}

// Run until we are only maxerr away from equilibrium.
// Typ. maxerr: 1e-7 (cannot go lower).
// Run for at most maxSteps to avoid infinite loop if we fail to relax.
//func (e *Heun) Relax(maxerr float64, maxSteps int) {
//	log.Println("relax down to", maxerr, "of equilibrium")
//	if maxerr < 1e-7 {
//		log.Fatal("relax: max error too small")
//	}
//	preverr := e.Maxerr
//	e.Maxerr = 1e-2
//
//	var i int
//	for i = 0; i < maxSteps; i++ {
//		e.Step()
//		if e.delta < e.Maxerr/e.Headroom {
//			e.Maxerr /= 2
//			e.dt_si /= 1.41
//		}
//		if e.Maxerr < maxerr {
//			break
//		}
//	}
//	if i == maxSteps {
//		log.Fatalf("relax: did not converge within %v time steps.")
//	}
//	e.Maxerr = preverr
//	util.DashExit()
//}
