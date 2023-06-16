/*
*
Based on the data provided, the most volatile 15-minute periods (the ones with highest ATR values) seem to be:

	Bucket 09:00: 16.604735
	Bucket 08:00: 15.042442
	Bucket 00:15: 14.061576
	Bucket 15:30: 14.160358
	Bucket 16:00: 14.092212

These periods appear to have the highest ATR values, indicating the highest volatility:

Early Morning (around 00:15 to 02:15): This period starts from 00:15, peaking at 00:15 with an ATR of 14.061576. It seems to decrease after 02:15.
Market Opening (around 07:00 to 10:00): This period begins from 07:00, reaching a peak at 09:00 with an ATR of 16.604735. After 10:00, the volatility appears to decrease.
Market Close (around 14:30 to 16:00): This period starts around 14:30, peaking at 15:30 with an ATR of 14.160358 and at 16:00 with an ATR of 14.092212. The volatility seems to start decreasing after 16:00.

Bucket 00:00: 0.000000
Bucket 00:15: 14.061576
Bucket 00:30: 8.418818
Bucket 00:45: 7.200499
Bucket 01:00: 8.888109
Bucket 01:15: 10.445361
Bucket 01:30: 10.200276
Bucket 01:45: 8.889157
Bucket 02:00: 8.342105
Bucket 02:15: 11.675262
Bucket 02:30: 8.518190
Bucket 02:45: 7.415408
Bucket 03:00: 7.864995
Bucket 03:15: 7.769359
Bucket 03:30: 8.453502
Bucket 03:45: 7.182053
Bucket 04:00: 6.811623
Bucket 04:15: 6.070685
Bucket 04:30: 6.188381
Bucket 04:45: 5.430496
Bucket 05:00: 6.155293
Bucket 05:15: 5.990393
Bucket 05:30: 6.155955
Bucket 05:45: 5.725982
Bucket 06:00: 6.796752
Bucket 06:15: 6.092542
Bucket 06:30: 6.723894
Bucket 06:45: 7.069163
Bucket 07:00: 11.766429
Bucket 07:15: 8.658015
Bucket 07:30: 8.962326
Bucket 07:45: 9.696419
Bucket 08:00: 15.042442
Bucket 08:15: 11.514030
Bucket 08:30: 10.953942
Bucket 08:45: 10.944156
Bucket 09:00: 16.604735
Bucket 09:15: 14.175951
Bucket 09:30: 13.152324
Bucket 09:45: 12.185654
Bucket 10:00: 12.184645
Bucket 10:15: 10.977556
Bucket 10:30: 10.592219
Bucket 10:45: 10.161866
Bucket 11:00: 10.225429
Bucket 11:15: 9.554077
Bucket 11:30: 9.243430
Bucket 11:45: 9.218897
Bucket 12:00: 9.366372
Bucket 12:15: 8.667020
Bucket 12:30: 8.591384
Bucket 12:45: 8.468802
Bucket 13:00: 9.083913
Bucket 13:15: 8.828064
Bucket 13:30: 9.819049
Bucket 13:45: 9.096338
Bucket 14:00: 9.120569
Bucket 14:15: 9.193123
Bucket 14:30: 13.071888
Bucket 14:45: 11.721134
Bucket 15:00: 11.900310
Bucket 15:15: 10.578388
Bucket 15:30: 14.160358
Bucket 15:45: 13.694555
Bucket 16:00: 14.092212
Bucket 16:15: 12.983108
Bucket 16:30: 12.630726
Bucket 16:45: 11.009886
Bucket 17:00: 10.574181
Bucket 17:15: 10.324006
Bucket 17:30: 9.741780
Bucket 17:45: 7.870619
Bucket 18:00: 7.555102
Bucket 18:15: 7.078723
Bucket 18:30: 6.855414
Bucket 18:45: 6.807732
Bucket 19:00: 7.222254
Bucket 19:15: 6.982798
Bucket 19:30: 7.020498
Bucket 19:45: 7.194645
Bucket 20:00: 7.986267
Bucket 20:15: 7.733421
Bucket 20:30: 8.114201
Bucket 20:45: 8.657132
Bucket 21:00: 6.734420
Bucket 21:15: 7.366325
Bucket 21:30: 8.216286
Bucket 21:45: 9.612563
Bucket 22:00: 0.527083
Bucket 22:15: 0.000000
Bucket 22:30: 0.000000
Bucket 22:45: 0.000000
Bucket 23:00: 0.000000
Bucket 23:15: 0.000000
Bucket 23:30: 0.000000
Bucket 23:45: 1.916667
*/
package main

import (
	"fmt"
	"time"

	"github.com/mwlazlo/srs/internal/pp"
	"gonum.org/v1/gonum/stat"
)

// Returns the bucket index for a given timestamp.
// Buckets are numbered from 0 to 95, corresponding to each 15-minute period in a 24-hour day.
func GetBucketIndex(timestamp time.Time) int {
	minutes := timestamp.Hour()*60 + timestamp.Minute()
	return minutes / 15
}

func CalculateBucketAtr(bars pp.Series) []float64 {
	bucketBars := make([][]*pp.Bar, 96)
	for i := 0; i < 96; i++ {
		bucketBars[i] = make([]*pp.Bar, 0)
	}

	// Split bars into buckets.
	for _, bar := range bars {
		bucket := GetBucketIndex(bar.Timestamp)
		bucketBars[bucket] = append(bucketBars[bucket], bar)
	}

	// Calculate ATR for each bucket.
	bucketAtr := make([]float64, 96)
	for i, bars := range bucketBars {
		if len(bars) > 0 {
			_, atr := pp.Atr(len(bars), bars)
			bucketAtr[i] = atr[len(atr)-1] // Take the last value as the ATR for this bucket.
		} else {
			bucketAtr[i] = 0
		}
	}

	return bucketAtr
}
func FindMostVolatileBucket(bars pp.Series) int {
	bucketAtr := CalculateBucketAtr(bars)
	maxAtr := float64(0)
	maxBucket := 0
	for i, atr := range bucketAtr {
		if atr > maxAtr {
			maxAtr = atr
			maxBucket = i
		}
	}
	return maxBucket
}

func PrintBucketAtr(bars pp.Series) {
	bucketAtr := CalculateBucketAtr(bars)
	for i, atr := range bucketAtr {
		// Convert bucket number back to a time.
		hours := i / 4
		minutes := (i % 4) * 15
		fmt.Printf("Bucket %02d:%02d: %f\n", hours, minutes, atr)
	}
}

func main() {
	bars, err := pp.ReadBarsFromCSV("data/dax-5m-fixed.csv", false)
	if err != nil {
		panic(err)
	}

	PrintBucketAtr(bars)
}

func dayOfWeek(bars []pp.Bar) {
	// Create a map to hold buckets for each hour
	buckets := make(map[time.Weekday][]float64)

	// Fill the buckets with close prices
	for _, bar := range bars {
		// The bucket index is the hour of the bar timestamp
		bucketIndex := bar.Timestamp.Weekday()
		buckets[bucketIndex] = append(buckets[bucketIndex], float64(bar.Close))
	}

	// Find the hour with the highest standard deviation
	var maxSD float64
	var maxSDDay int
	for i := 0; i < 7; i++ {
		sd := stat.StdDev(buckets[time.Weekday(i)], nil)
		if sd > maxSD {
			maxSD = sd
			maxSDDay = i
		}
		fmt.Printf("%s, SD = %v\n",
			time.Weekday(i).String(), sd)
	}

	fmt.Printf("Day of week with the highest standard deviation: %s, SD = %v\n",
		time.Weekday(maxSDDay), maxSD)
}

func hourly(bars pp.Series) {
	// Create a map to hold buckets for each hour
	buckets := make(map[int][]float64)

	// Fill the buckets with close prices
	for _, bar := range bars {
		// The bucket index is the hour of the bar timestamp
		bucketIndex := bar.Timestamp.Hour()
		if bucketIndex == 14 {
			//runtime.Breakpoint()
		}
		buckets[bucketIndex] = append(buckets[bucketIndex], float64(bar.Close))
	}

	for i := 0; i < len(buckets); i++ {
		fmt.Printf("%d: %d\n", i, len(buckets[i]))
	}

	// Find the hour with the highest standard deviation
	var maxSD float64
	var maxSDHour int
	for i := 0; i < 24; i++ {
		sd := stat.StdDev(buckets[i], nil)
		if sd > maxSD {
			maxSD = sd
			maxSDHour = i
		}
		fmt.Printf("%2d %02d:00 to %02d:59, SD = %v\n",
			len(buckets[i]), i, i+1, sd)
	}

	fmt.Printf("Hour with the highest standard deviation: %02d:00 to %02d:59, SD = %v\n",
		maxSDHour, maxSDHour, maxSD)
}

func fiveMinute(bars []pp.Bar) {
	// Create a map to hold 12 buckets for each 5-minute interval of an hour
	buckets := make(map[int][]float64)

	// Fill the buckets with close prices
	for _, bar := range bars {
		// The bucket index is the minute of the bar timestamp divided by 5
		bucketIndex := bar.Timestamp.Minute() / 5
		buckets[bucketIndex] = append(buckets[bucketIndex], float64(bar.Close))
	}

	// Find the 5-minute interval with the highest standard deviation
	var maxSD float64
	var maxSDInterval int
	for i := 0; i < 12; i++ {
		sd := stat.StdDev(buckets[i], nil)
		if sd > maxSD {
			maxSD = sd
			maxSDInterval = i
		}
		fmt.Printf("%02d:%02d to %02d:%02d, SD = %v\n",
			i*5, 0, (i+1)*5, 0, sd)

	}

	fmt.Printf("5-minute interval with the highest standard deviation: %02d:%02d to %02d:%02d, SD = %v\n",
		maxSDInterval*5, 0, (maxSDInterval+1)*5, 0, maxSD)
}
