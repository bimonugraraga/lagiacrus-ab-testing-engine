package abtestingengine

import "fmt"

type Variant struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Weight int    `json:"weight"`
}

type Experiment struct {
	ID       string    `json:"id"`
	Variants []Variant `json:"variants"`
}

type ResultAssignBulk struct {
	ResultsBulk map[string]ResultBulk
}
type ResultBulk struct {
	UserID  []string
	Variant Variant
}

type UserConversion struct {
	UserID       any
	ExperimentID any
	VariantID    any
}

type ResultAnalytics struct {
	ExperimentID   string
	Variants       []Variant
	Exposure       []map[any]any
	Conversion     []map[any]any
	ConversionRate []map[any]any
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

func GetListOfAllKeysFormat(experimentID any) []string {
	return []string{
		keyGeneratorUserExposureLock(experimentID),
		keyGeneratorUserExposureAnalytics(experimentID),
		keyGeneratorUserConversionLock(experimentID),
		keyGeneratorUserConversionAnalytics(experimentID),
	}
}
