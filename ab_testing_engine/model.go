package abtestingengine

import "fmt"

type Variant struct {
	ID     string
	Name   string
	Weight int
}

type Experiment struct {
	ID       string
	Variants []Variant
}

// type ResultBulkInt struct {
// 	UserID  []int64
// 	Variant VariantInt
// }

type UserConversion struct {
	UserID       any
	ExperimentID any
	VariantID    any
}

type Analytics struct {
	ExperimentID string
}

type ResultAnalytics struct {
	ExperimentID   string
	Variants       []Variant
	Exposure       map[any]int
	Conversion     map[any]int
	ConversionRate map[any]float64
	ExpiredOnLocal string
	ExpiredOnUTC   string
}

func keyGeneratorExperimentDetail(experimentID any) string {
	return fmt.Sprintf("lagiacrus:experiment:{%v}", experimentID)
}
func keyGeneratorExpiredTime(experimentID any) string {
	return fmt.Sprintf("lagiacrus:experiment:{%v}:expired_time", experimentID)
}

func keyGeneratorUserExposureLock(experimentID any) string {
	return fmt.Sprintf("lagiacrus:experiment:{%v}:exposure_lock", experimentID)
}

func keyGeneratorUserExposureAnalytics(experimentID any) string {
	return fmt.Sprintf("lagiacrus:experiment:{%v}:exposure", experimentID)
}

func keyGeneratorUserConversionLock(experimentID any) string {
	return fmt.Sprintf("lagiacrus:experiment:{%v}:conversion_lock", experimentID)
}

func keyGeneratorUserConversionAnalytics(experimentID any) string {
	return fmt.Sprintf("lagiacrus:experiment:{%v}:conversion", experimentID)
}
