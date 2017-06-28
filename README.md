### HLL Check

This is a small program that can be used to compare two HLL / HLL++ implementations to each other across a set of different cardinality and duplicated data.

At the moment it just generates different data sets and compares the accuracy between two one or more implementations.
In the future I plan to provide benchmarks, which can be run on competing implementations, and more analysis of how errors change over time (as a dataset grows).

#### Using the package

To use the package you need to write a small `main` program, which provides `hllcheck` with one or two factory functions for initialising new HLL/HLL++ implementations.

A HLL/HLL++ implementation must satisfy the following interface:

```go
type HLL interface {
	Add(v []byte)
	Count() uint64
}
```

A simple main program could look like:

```go
package main

import (
	"os"

	"github.com/other/repo/hll2"

	"github.com/influxdata/hll-check"
	"github.com/influxdata/influxdb/pkg/estimator/hll"
)

func main() {
	hllcheck.Seed = time.Now().Unix()
	// Existing implementation with precision 16.
	h1f := func() hllcheck.HLL { return hll.MustNewPlus(16) }
	// Proposed alternative implementation with precision 16.
	h2f := func() hllcheck.HLL { return hll2.New(16) }

	_ = hllcheck.Run(hllcheck.ToHLLFatory(h1f), hllcheck.ToHLLFatory(h2f), os.Stdout)
}
```

In this case the results will be printed to `stdout`. You could not ignore the returned value and instead inspect or do further analysis on the results yourself.


The current results for the version of HLL++ in InfluxDB `1.3` are:

```
Size			Actual Cardinality	Estimation		Error (%)		Duplication (%)
500			500			500			0.0000%			0.00%
500			359			360			0.2778%			28.20%
500			112			113			0.8850%			77.60%
1000			1000			1000			0.0000%			0.00%
1000			756			756			0.0000%			24.40%
1000			199			200			0.5000%			80.10%
5000			5000			5000			0.0000%			0.00%
5000			3743			3743			0.0000%			25.14%
5000			1005			1005			0.0000%			79.90%
10000			10000			10000			0.0000%			0.00%
10000			7467			7467			0.0000%			25.33%
10000			1976			1977			0.0506%			80.24%
100000			100000			100123			0.1228%			0.00%
100000			74895			74973			0.1040%			25.11%
100000			19895			19894			-0.0050%		80.11%
250000			250000			249427			-0.2297%		0.00%
250000			187589			187566			-0.0123%		24.96%
250000			50072			49783			-0.5805%		79.97%
500000			500000			499968			-0.0064%		0.00%
500000			374736			375053			0.0845%			25.05%
500000			100410			100534			0.1233%			79.92%
1000000			1000000			1002466			0.2460%			0.00%
1000000			749999			749239			-0.1014%		25.00%
1000000			200620			200144			-0.2378%		79.94%
5000000			5000000			5009290			0.1855%			0.00%
5000000			3749466			3760525			0.2941%			25.01%
5000000			1000467			1003146			0.2671%			79.99%
25000000		25000000		24994384		-0.0225%		0.00%
25000000		18749778		18721102		-0.1532%		25.00%
25000000		5001941			5011561			0.1920%			79.99%
100000000		100000000		99684277		-0.3167%		0.00%
100000000		75001559		75072214		0.0941%			25.00%
500000000		500000000		500475904		0.0951%			0.00%
500000000		374990807		374916524		-0.0198%		25.00%



Mean Error		Median Error		Error Variance		Max Error
0.0540%			0.0000%			0.0574			0.8850%
```
