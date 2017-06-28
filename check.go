package hllcheck

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/rand"
	"text/tabwriter"

	"github.com/e-dard/godist"
)

// Seed sets the seed used by the `math/rand` package.
var Seed = int64(1)

var params = []struct {
	n uint64
	p float64
}{
	// Low cardinality with varying levels of duplication (density).
	{n: 500, p: 0.0}, {n: 500, p: 0.25}, {n: 500, p: 0.8},

	// Still small cardinality with varying levels of duplication.
	{n: 1000, p: 0.0}, {n: 1000, p: 0.25}, {n: 1000, p: 0.8},
	{n: 5000, p: 0.0}, {n: 5000, p: 0.25}, {n: 5000, p: 0.8},
	{n: 10000, p: 0.0}, {n: 10000, p: 0.25}, {n: 10000, p: 0.8},

	//  Medium cardinality with varying levels of duplication.
	{n: 100000, p: 0.0}, {n: 100000, p: 0.25}, {n: 100000, p: 0.8},
	{n: 250000, p: 0.0}, {n: 250000, p: 0.25}, {n: 250000, p: 0.8},
	{n: 500000, p: 0.0}, {n: 500000, p: 0.25}, {n: 500000, p: 0.8},

	// Higher cardinality
	{n: 1000000, p: 0.0}, {n: 1000000, p: 0.25}, {n: 1000000, p: 0.8},
	{n: 5000000, p: 0.0}, {n: 5000000, p: 0.25}, {n: 5000000, p: 0.8},
	{n: 25000000, p: 0.0}, {n: 25000000, p: 0.25}, {n: 25000000, p: 0.8},

	// Very high
	{n: 100000000, p: 0.0}, {n: 100000000, p: 0.25},
	{n: 500000000, p: 0.0}, {n: 500000000, p: 0.25},
}

// RunData describes a set of data used for a single test run. RunData is shaped
// via the n and p parameters. n represents the size of the set, and p determines
// the probability of a duplicate value being inserted.
type RunData struct {
	n uint64
	p float64

	i           uint64 // Current progress of iterator.
	x           uint64 // Current value in itereator.
	cardinality uint64
}

// NewRunData initialises a new RunData for evaluating the performance of a HLL++
// implementation.
func NewRunData(n uint64, p float64) *RunData {
	if p < 0.0 || p > 1.0 {
		panic(fmt.Sprintf("invalid p value %v", p))
	}
	return &RunData{n: n, p: p}
}

// Next returns the next value in the RunData. The second returned value
// determines if the iterator is complete.
func (r *RunData) Next() (uint64, bool) {
	if r.i == r.n {
		return 0, false
	}
	r.i++

	if rand.Float64() >= r.p { // Move to next value and increase cardinality
		r.x++
		r.cardinality++
	}
	return r.x, true
}

// Cardinality returns the cardinality of the RunData dataset. The true
// cardinality is not known until the RunData's iterator has been drained.
// Cardinality panics if it's called prior to that.
func (r *RunData) Cardinality() uint64 {
	if r.i < r.n {
		panic("cardinality not available until all values have been generated")
	}
	return r.cardinality
}

// Size returns the size of the RunData dataset.
func (r *RunData) Size() uint64 {
	return r.n
}

// Results contains a collection of results.
type Results [][]Result

// Result describes the result of an implementation on a dataset.
type Result struct {
	ActualCardinality    uint64
	EstimatedCardinality uint64
	Size                 uint64
}

// ErrorPercent returns the marginal error as a percentage between the estimated
// cardinality and the actual cardinaltiy. A result of -0.76 for example implies
// and error of -0.76% (the estimated cardinality was less than one percent the
// actual under the actual cardinality value).
func (r Result) ErrorPercent() float64 {
	actual := float64(r.ActualCardinality)
	estimated := float64(r.EstimatedCardinality)
	return 100.0 * (1 - (actual / estimated))
}

// Duplication calculates how many values (as a proportion of all added values)
// were duplicates.
func (r Result) Duplication() float64 {
	return 100.0 * (float64(r.Size-r.ActualCardinality) / float64(r.Size))
}

// HLL is the interface that a HLL/HLL++ algorithm must implement.
type HLL interface {
	Add(v []byte)
	Count() uint64
}

// HLLFactory is the interface that generates new instantiations of HLL implementations.
type HLLFactory interface {
	New() HLL
}

type hllFactory struct {
	f func() HLL
}

func (h hllFactory) New() HLL {
	return h.f()
}

// ToHLLFatory is a helper for converting a function that returns an HLL
// implementation to an HLL Factory type.
func ToHLLFatory(f func() HLL) HLLFactory {
	return hllFactory{f: f}
}

// Run runs a suite of tests against one or more HLL implementations, focussing
// on the accuracy of the algorithms. If w is non-nil, then the results will be
// written there as well as returned.
func Run(h1, h2 HLLFactory, w io.Writer) Results {
	rand.Seed(Seed) // Set the seed.

	if h1 == nil {
		panic("must provide at least one implementation")
	}

	var tw *tabwriter.Writer
	if w != nil {
		tw = tabwriter.NewWriter(w, 24, 8, 4, '\t', 0)
	}

	var (
		factories = []HLLFactory{h1, h2}
		results   = make(Results, 2)
		buf       = make([]byte, 8)
	)

	if tw != nil {
		fmt.Fprint(tw, "Size\tActual Cardinality\tEstimation\tError (%)\tDuplication (%)\n")
	}

	for _, param := range params {

		rd := NewRunData(param.n, param.p)

		h1 := factories[0].New()
		var h2 HLL
		if factories[1] != nil {
			h2 = factories[1].New()
		}

		for {
			v, ok := rd.Next()
			if !ok {
				break
			}
			binary.BigEndian.PutUint64(buf, v)
			h1.Add(buf)
			if h2 != nil {
				h2.Add(buf)
			}
		}

		result1 := Result{
			ActualCardinality:    rd.Cardinality(),
			EstimatedCardinality: h1.Count(),
			Size:                 rd.Size(),
		}
		results[0] = append(results[0], result1)

		var result2 Result
		if h2 != nil {
			result2 = Result{
				ActualCardinality:    rd.Cardinality(),
				EstimatedCardinality: h2.Count(),
				Size:                 rd.Size(),
			}
			results[1] = append(results[1], result2)
		}

		if tw != nil {
			fmt.Fprintf(tw, "%d\t%d\t%d\t%0.4f%%\t%0.2f%%\n", result1.Size, result1.ActualCardinality, result1.EstimatedCardinality, result1.ErrorPercent(), result1.Duplication())
			if h2 != nil {
				fmt.Fprintf(tw, "%d\t%d\t%d\t%0.4f%%\t%0.2f%%\n", result2.Size, result2.ActualCardinality, result2.EstimatedCardinality, result2.ErrorPercent(), result2.Duplication())
			}
		}

		if tw != nil {
			fmt.Fprint(tw, "\n")
			tw.Flush()
		}

	}

	if tw == nil {
		return results
	}

	fmt.Fprint(tw, "\n\nMean Error\tMedian Error\tError Variance\tMax Error\n")
	// Analyse results.
	for i, factory := range factories {
		if factory == nil {
			continue
		}

		var (
			maxError    float64
			meanError   float64
			medianError float64
			variance    float64
		)

		dist := godist.Empirical{}
		for _, result := range results[i] {
			err := result.ErrorPercent()
			if math.Abs(err) > math.Abs(maxError) {
				maxError = err
			}
			dist.Add(err)
		}

		meanError, _ = dist.Mean()
		medianError, _ = dist.Median()
		variance, _ = dist.Variance()
		fmt.Fprintf(tw, "%0.4f%%\t%0.4f%%\t%0.4f\t%0.4f%%\t\n", meanError, medianError, variance, maxError)
	}
	tw.Flush()
	return results
}
