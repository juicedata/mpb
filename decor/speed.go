package decor

import (
	"fmt"
	"io"
	"math"
	"time"

	"github.com/VividCortex/ewma"
)

var (
	_ Decorator        = (*movingAverageSpeed)(nil)
	_ EwmaDecorator    = (*movingAverageSpeed)(nil)
	_ Decorator        = (*averageSpeed)(nil)
	_ AverageDecorator = (*averageSpeed)(nil)
)

// FmtAsSpeed adds "/s" to the end of the input formatter. To be
// used with SizeB1000 or SizeB1024 types, for example:
//
//	fmt.Printf("%.1f", FmtAsSpeed(SizeB1024(2048)))
func FmtAsSpeed(input fmt.Formatter) fmt.Formatter {
	return speedFormatter{input}
}

type speedFormatter struct {
	fmt.Formatter
}

func (self speedFormatter) Format(st fmt.State, verb rune) {
	self.Formatter.Format(st, verb)
	_, err := io.WriteString(st, "/s")
	if err != nil {
		panic(err)
	}
}

// EwmaSpeed exponential-weighted-moving-average based speed decorator.
// For this decorator to work correctly you have to measure each iteration's
// duration and pass it to one of the (*Bar).EwmaIncr... family methods.
func EwmaSpeed(unit interface{}, format string, age float64, wcc ...WC) Decorator {
	var average ewma.MovingAverage
	if age == 0 {
		average = ewma.NewMovingAverage()
	} else {
		average = ewma.NewMovingAverage(age)
	}
	return MovingAverageSpeed(unit, format, NewThreadSafeMovingAverage(average), wcc...)
}

// MovingAverageSpeed decorator relies on MovingAverage implementation
// to calculate its average.
//
//	`unit` one of [0|SizeB1024(0)|SizeB1000(0)]
//
//	`format` printf compatible verb for value, like "%f" or "%d"
//
//	`average` MovingAverage implementation
//
//	`wcc` optional WC config
//
// format examples:
//
//	unit=SizeB1024(0), format="%.1f"  output: "1.0MiB/s"
//	unit=SizeB1024(0), format="% .1f" output: "1.0 MiB/s"
//	unit=SizeB1000(0), format="%.1f"  output: "1.0MB/s"
//	unit=SizeB1000(0), format="% .1f" output: "1.0 MB/s"
func MovingAverageSpeed(unit interface{}, format string, average ewma.MovingAverage, wcc ...WC) Decorator {
	if format == "" {
		format = "% d"
	}
	d := &movingAverageSpeed{
		WC:       initWC(wcc...),
		average:  average,
		producer: chooseSpeedProducer(unit, format),
	}
	return d
}

type movingAverageSpeed struct {
	WC
	producer func(float64) string
	average  ewma.MovingAverage
	msg      string
}

func (d *movingAverageSpeed) Decor(s Statistics) string {
	if !s.Completed {
		var speed float64
		if v := d.average.Value(); v > 0 {
			speed = 1 / v
		}
		d.msg = d.producer(speed * 1e9)
	}
	return d.FormatMsg(d.msg)
}

func (d *movingAverageSpeed) EwmaUpdate(n int64, dur time.Duration) {
	durPerByte := float64(dur) / float64(n)
	if math.IsInf(durPerByte, 0) || math.IsNaN(durPerByte) {
		return
	}
	d.average.Add(durPerByte)
}

// AverageSpeed decorator with dynamic unit measure adjustment. It's
// a wrapper of NewAverageSpeed.
func AverageSpeed(unit int, format string, wcc ...WC) Decorator {
	return NewAverageSpeed(unit, format, time.Now(), wcc...)
}

// NewAverageSpeed decorator with dynamic unit measure adjustment and
// user provided start time.
//
//	`unit` one of [0|SizeB1024(0)|SizeB1000(0)]
//
//	`format` printf compatible verb for value, like "%f" or "%d"
//
//	`startTime` start time
//
//	`wcc` optional WC config
//
// format examples:
//
//	unit=SizeB1024(0), format="%.1f"  output: "1.0MiB/s"
//	unit=SizeB1024(0), format="% .1f" output: "1.0 MiB/s"
//	unit=SizeB1000(0), format="%.1f"  output: "1.0MB/s"
//	unit=SizeB1000(0), format="% .1f" output: "1.0 MB/s"
func NewAverageSpeed(unit interface{}, format string, startTime time.Time, wcc ...WC) Decorator {
	if format == "" {
		format = "% d"
	}
	d := &averageSpeed{
		WC:        initWC(wcc...),
		startTime: startTime,
		producer:  chooseSpeedProducer(unit, format),
	}
	return d
}

type averageSpeed struct {
	WC
	startTime time.Time
	producer  func(float64) string
	msg       string
}

func (d *averageSpeed) Decor(s Statistics) string {
	if !s.Completed {
		speed := float64(s.Current) / float64(time.Since(d.startTime))
		d.msg = d.producer(speed * 1e9)
	}
	return d.FormatMsg(d.msg)
}

func (d *averageSpeed) AverageAdjust(startTime time.Time) {
	d.startTime = startTime
}

func chooseSpeedProducer(unit interface{}, format string) func(float64) string {
	switch unit.(type) {
	case SizeB1024:
		return func(speed float64) string {
			return fmt.Sprintf(format, FmtAsSpeed(SizeB1024(math.Round(speed))))
		}
	case SizeB1000:
		return func(speed float64) string {
			return fmt.Sprintf(format, FmtAsSpeed(SizeB1000(math.Round(speed))))
		}
	default:
		return func(speed float64) string {
			return fmt.Sprintf(format, speed)
		}
	}
}
