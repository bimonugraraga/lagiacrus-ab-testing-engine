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
	UserID       string
	ExperimentID string
	VariantID    string
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

func keyGeneratorExperimentDetail(experimentID string) string {
	return fmt.Sprintf("lagiacrus:experiment:{%s}", experimentID)
}
func keyGeneratorExpiredTime(experimentID string) string {
	return fmt.Sprintf("lagiacrus:experiment:{%s}:expired_time", experimentID)
}

func keyGeneratorUserExposureLock(experimentID string) string {
	return fmt.Sprintf("lagiacrus:experiment:{%s}:exposure_lock", experimentID)
}

func keyGeneratorUserExposureAnalytics(experimentID string) string {
	return fmt.Sprintf("lagiacrus:experiment:{%s}:exposure", experimentID)
}

func keyGeneratorUserConversionLock(experimentID string) string {
	return fmt.Sprintf("lagiacrus:experiment:{%s}:conversion_lock", experimentID)
}

func keyGeneratorUserConversionAnalytics(experimentID any) string {
	return fmt.Sprintf("lagiacrus:experiment:{%v}:conversion", experimentID)
}

func GetListOfAllKeysFormat(experimentID string) []string {
	return []string{
		keyGeneratorUserExposureLock(experimentID),
		keyGeneratorUserExposureAnalytics(experimentID),
		keyGeneratorUserConversionLock(experimentID),
		keyGeneratorUserConversionAnalytics(experimentID),
	}
}
