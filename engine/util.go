package engine

import (
	"fmt"
	"github.com/mumax/3/cuda"
	"github.com/mumax/3/data"
	"github.com/mumax/3/dump"
	"github.com/mumax/3/mag"
	"github.com/mumax/3/oommf"
	"github.com/mumax/3/util"
	"math"
	"os"
	"path"
	"sort"
	"strings"
)

func init() {
	DeclFunc("Expect", Expect, "Used for automated tests: checks if a value is close enough to the expected value")
	DeclFunc("ExpectV", ExpectV, "Used for automated tests: checks if a vector is close enough to the expected value")
	DeclFunc("Fprintln", Fprintln, "Print to file")
	DeclFunc("Sign", sign, "Signum function")
	DeclFunc("Vector", Vector, "Constructs a vector with given components")
	DeclConst("Mu0", mag.Mu0, "Permittivity of vaccum (Tm/A)")
	DeclFunc("Print", myprint, "Print to standard output")
	DeclFunc("LoadFile", LoadFile, "Load a data file (ovf or dump)")
	DeclFunc("Index2Coord", Index2Coord, "Convert cell index to x,y,z coordinate in meter")
	DeclFunc("NewSlice", NewSlice, "Makes a 3D array of scalars with given x,y,z size")
}

// Returns a new new slice (3D array) with given number of components and size.
func NewSlice(ncomp, Nx, Ny, Nz int) *data.Slice {
	return data.NewSlice(ncomp, [3]int{Nx, Ny, Nz})
}

// Constructs a vector
func Vector(x, y, z float64) data.Vector {
	return data.Vector{x, y, z}
}

// Test if have lies within want +/- maxError,
// and print suited message.
func Expect(msg string, have, want, maxError float64) {
	if math.IsNaN(have) || math.IsNaN(want) || math.Abs(have-want) > maxError {
		util.Fatal(msg, ":", " have: ", have, " want: ", want, "±", maxError)
	} else {
		LogOutput(msg, ":", have, "OK")
	}
	// note: we also check "want" for NaN in case "have" and "want" are switched.
}

func ExpectV(msg string, have, want data.Vector, maxErr float64) {
	for c := 0; c < 3; c++ {
		Expect(fmt.Sprintf("%v[%v]", msg, c), have[c], want[c], maxErr)
	}
}

// Append msg to file. Used to write aggregated output of many simulations in one file.
func Fprintln(filename string, msg ...interface{}) {
	if !path.IsAbs(filename) {
		filename = OD + "/" + filename
	}
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	util.FatalErr(err)
	defer f.Close()
	_, err = fmt.Fprintln(f, myFmt(msg)...)
	util.FatalErr(err)
}

// Read a magnetization state from .dump file.
func LoadFile(fname string) *data.Slice {
	if path.Ext(fname) == ".dump" {
		s, _ := dump.MustReadFile(fname)
		return s
	} else {
		s, _ := oommf.MustReadFile(fname)
		return s
	}
}

// Download a quantity to host,
// or just return its data when already on host.
func Download(q Quantity) *data.Slice {
	buf, recycle := q.Slice()
	if recycle {
		defer cuda.Recycle(buf)
	}
	if buf.CPUAccess() {
		return buf
	} else {
		return buf.HostCopy()
	}
}

// print with special formatting for some known types
func myprint(msg ...interface{}) {
	LogOutput(myFmt(msg)...)
}

// mumax specific formatting (Slice -> average, etc).
func myFmt(msg []interface{}) []interface{} {
	for i, m := range msg {
		if e, ok := m.(*float64); ok {
			msg[i] = *e
		}
		// Tabledata: print average
		if m, ok := m.(TableData); ok {
			str := fmt.Sprint(m.average())
			msg[i] = str[1 : len(str)-1] // remove [ ]
			continue
		}
	}
	return msg
}

// converts cell index to coordinate, internal coordinates
func Index2Coord(ix, iy, iz int) data.Vector {
	m := Mesh()
	n := m.Size()
	c := m.CellSize()
	x := c[X]*(float64(ix)-0.5*float64(n[X]-1)) - TotalShift
	y := c[Y] * (float64(iy) - 0.5*float64(n[Y]-1))
	z := c[Z] * (float64(iz) - 0.5*float64(n[Z]-1))
	return data.Vector{x, y, z}
}

func sign(x float64) float64 {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	default:
		return 0
	}
}

// returns a/b, or 0 when b == 0
func safediv(a, b float32) float32 {
	if b == 0 {
		return 0
	} else {
		return a / b
	}
}

// shortcut for slicing unaddressable_vector()[:]
func slice(v [3]float64) []float64 {
	return v[:]
}

func unslice(v []float64) [3]float64 {
	util.Assert(len(v) == 3)
	return [3]float64{v[0], v[1], v[2]}
}

type caseIndep []string

func (s *caseIndep) Len() int           { return len(*s) }
func (s *caseIndep) Less(i, j int) bool { return strings.ToLower((*s)[i]) < strings.ToLower((*s)[j]) }
func (s *caseIndep) Swap(i, j int)      { (*s)[i], (*s)[j] = (*s)[j], (*s)[i] }

func sortNoCase(s []string) {
	i := caseIndep(s)
	sort.Sort(&i)
}

// trim trailing newlines
func rmln(a string) string {
	for strings.HasSuffix(a, "\n") {
		a = a[:len(a)-1]
	}
	return a
}

const (
	X = 0
	Y = 1
	Z = 2
)

const (
	SCALAR = 1
	VECTOR = 3
)
